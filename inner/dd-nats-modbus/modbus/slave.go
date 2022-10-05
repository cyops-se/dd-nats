package modbus

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	mb "github.com/simonvetter/modbus"
)

type modbusDataset struct {
	start uint16
	count uint16
	fc    uint16
	items []*ModbusItem
	err   error
}

type modbusConnection struct {
	client    *mb.ModbusClient
	Connected bool             `json:"connected"` // Open() successful
	Slave     *ModbusSlaveItem `json:"slave"`
	Datasets  []*modbusDataset `json:"datasets"`
	Abort     bool             `json:"abort"`
	ErrStr    string           `json:"error"`
	Err       error
}

type ItemByAddress []*ModbusItem

var modbusConnections []*modbusConnection
var slaves []*ModbusSlaveItem
var datapoints []*ModbusItem
var engineLock sync.Mutex

var TRACE bool

func (a ItemByAddress) Len() int           { return len(a) }
func (a ItemByAddress) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ItemByAddress) Less(i, j int) bool { return a[i].ModbusAddress < a[j].ModbusAddress }

func transposeAddress(item *ModbusItem) uint16 {
	if item.FunctionCode == 3 && item.ModbusAddress >= 40000 {
		item.AdaptedAddress = item.ModbusAddress - 40000
	} else if item.FunctionCode == 4 && item.ModbusAddress >= 30000 {
		item.AdaptedAddress = item.ModbusAddress - 30000
	}

	return uint16(item.AdaptedAddress)
}

func buildDatasets(slaves []*ModbusSlaveItem, allitems []*ModbusItem) {
	log.Printf("Building datasets ...")

	for _, slave := range slaves {
		if TRACE {
			log.Printf(" ... for slave: %s, ip: %s, port: %d, offset: %d", slave.Name, slave.IPAddress, slave.Port, slave.Offset)
		}

		mc := &modbusConnection{Slave: slave}
		items := findItemsForSlave(slave, allitems)
		if items == nil || len(items) <= 0 {
			log.Printf("No items available for modbus slave %s", mc.Slave.IPAddress)
			continue
		}

		sort.Sort(ItemByAddress(items))

		ds := &modbusDataset{}
		ds.fc = uint16(items[0].FunctionCode)
		ds.start = uint16(int(transposeAddress(items[0])) + slave.Offset)
		ds.count = uint16(len(items))
		ds.items = items

		startindex := 0
		for n, item := range items {
			if n-startindex > 0 && item.ModbusAddress-items[n-1].ModbusAddress > 1 {
				log.Printf("n: %d, startindex: %d", n, startindex)
				ds.count = uint16(n - startindex)
				ds.items = items[startindex:n]
				mc.Datasets = append(mc.Datasets, ds)

				startindex = n
				ds = &modbusDataset{}
				ds.fc = uint16(items[startindex].FunctionCode)
				ds.start = uint16(int(transposeAddress(items[startindex])) + slave.Offset)
				ds.count = uint16(len(items) - startindex)
				ds.items = items[startindex:]
			}
		}

		mc.Datasets = append(mc.Datasets, ds)

		if TRACE {
			for dno, ds := range mc.Datasets {
				log.Printf("ds: %d, start: %d, count: %d, len(items): %d, offset: %d", dno, ds.start, ds.count, len(ds.items), slave.Offset)
				for i, item := range ds.items {
					log.Printf("item: %d, name: %s", i, item.Name)
				}
			}
		}

		mc.Slave.State = ModbusSlaveStateRunning
		modbusConnections = append(modbusConnections, mc)
	}

	log.Printf("Done building datasets for %d Modbus slaves!", len(slaves))
}

func findItemsForSlave(slave *ModbusSlaveItem, allitems []*ModbusItem) []*ModbusItem {
	result := make([]*ModbusItem, 0)
	for _, item := range allitems {
		if item.ModbusSlaveID == slave.ID {
			result = append(result, item)
		}
	}
	return result
}

func (mc *modbusConnection) checkConnection() error {
	if mc.client == nil {
		mc.client, mc.Err = mb.NewClient(&mb.ClientConfiguration{
			URL:     fmt.Sprintf("tcp://%s:%d", mc.Slave.IPAddress, mc.Slave.Port),
			Timeout: 1 * time.Second,
		})
	}

	if !mc.Connected {
		if mc.Err = mc.client.Open(); mc.Err == nil {
			mc.Connected = true
			logger.Trace("Modbus service", "Modbus client connection successful for slave: %s", mc.Slave.IPAddress)
			mc.ErrStr = ""
			mc.Slave.ErrorMsg = ""
		} else {
			mc.ErrStr = mc.Err.Error()
			mc.Slave.LastError = time.Now().UTC().Format("2006-01-02 15:04:05")
			mc.Slave.ErrorMsg = mc.Err.Error()
		}
	}

	return mc.Err
}

func (mc *modbusConnection) closeConnection() {
	if mc != nil && mc.client != nil && mc.Connected {
		mc.Connected = false
		mc.client.Close()
		mc.client = nil
	}
}

// Function code 3 = Holding registers, 4 = Input registers
// NOTE! This is a very limited implementation right now that only support 16 bit uint values
func (mc *modbusConnection) runSlaveWorker() {
	timer := time.NewTicker(1 * time.Second)

	for {
		if mc.Slave.State != ModbusSlaveStateStopped {
			// engineLock.Lock()

			mc.checkConnection()
			if mc.Connected {
				for dsno, ds := range mc.Datasets {
					var msg types.DataPointSample

					var rawvalues []uint16
					if ds.fc == 3 {
						rawvalues, mc.Err = mc.client.ReadRegisters(ds.start, ds.count, mb.HOLDING_REGISTER)
					} else if ds.fc == 4 {
						rawvalues, mc.Err = mc.client.ReadRegisters(ds.start, ds.count, mb.INPUT_REGISTER)
					} else {
						mc.Err = fmt.Errorf("Modbus function code not supported: %d", ds.fc)
						mc.Slave.LastError = time.Now().UTC().Format("2006-01-02 15:04:05")
						mc.Slave.ErrorMsg = mc.Err.Error()
					}

					if mc.Err != nil {
						log.Printf("Failure to read registers on modbus slave %s, start: %d, count: %d, error: %s",
							mc.Slave.IPAddress, ds.start, ds.count, mc.Err.Error())

						mc.ErrStr = mc.Err.Error()
						mc.Slave.LastError = time.Now().UTC().Format("2006-01-02 15:04:05")
						mc.Slave.ErrorMsg = mc.Err.Error()
						mc.closeConnection()
						break
					}

					// There should be as many values as there are items in the dataset
					msg.Points = make([]types.DataPoint, len(rawvalues))
					msg.Group = fmt.Sprintf("%s_%d", mc.Slave.IPAddress, dsno)
					for n, rawvalue := range rawvalues {
						if n < len(ds.items) {
							item := ds.items[n]
							factor := float64(rawvalue-item.PlcRangeMin) / float64(item.PlcRangeMax-item.PlcRangeMin)
							value := float64(item.RangeMax-item.RangeMin)*float64(factor) + float64(item.RangeMin)
							quality := 192 // OPC quality GOOD (non-specific)
							if rawvalue < item.PlcRangeMin || rawvalue > item.PlcRangeMax {
								quality = 0 // OPC quality BAD (non-specific)
							}

							msg.Points[n].Time = time.Now().UTC()
							msg.Points[n].Quality = quality
							msg.Points[n].Name = ds.items[n].Name
							msg.Points[n].Value = value

							if TRACE {
								log.Printf("ds: %d, name: %s, address: %d, raw value: %v, value: %f", dsno, item.Name, int(ds.start)+n, rawvalue, value)
								log.Printf("msg.Time: %v, msg.Name: %s, msg.Value: %v, msg.Quality: %d",
									msg.Points[n].Time, msg.Points[n].Name, msg.Points[n].Value, msg.Points[n].Quality)
							}
						}
					}

					ddnats.Publish("forward.process", msg)

					mc.Slave.LastRun = time.Now().UTC().Format("2006-01-02 15:04:05")
				}
			}
			// engineLock.Unlock()
		}

		<-timer.C
		if mc.Abort {
			mc.closeConnection()
			break
		}
	}
}

func abortExistingConnections() {
	for _, mc := range modbusConnections {
		mc.Abort = true
	}

	// Allow the workers to exit
	if len(modbusConnections) > 0 {
		time.Sleep(5 * time.Second)
	}

	modbusConnections = nil
}

func getConnectionForSlave(slave *ModbusSlaveItem) *modbusConnection {
	for _, mc := range modbusConnections {
		if mc.Slave != nil && mc.Slave.ID == slave.ID {
			return mc
		}
	}

	return nil
}

func getSlaveByIP(ip string) *ModbusSlaveItem {
	for _, slave := range slaves {
		if slave.IPAddress == ip {
			return slave
		}
	}

	return nil
}

func checkSlaveIP(ip string) (uint, error) {
	if strings.TrimSpace(ip) == "" {
		return 0, logger.Error("Modbus service", "Failed to check empty modbus slave IP")
	}

	var item ModbusSlaveItem
	err := db.DB.First(&item, "ip_address = ?", ip).Error
	if err != nil {
		item.IPAddress = ip
		item.Name = ip
		AddModbusSlave(&item)
		log.Println("Creating ip:", ip)
		return checkSlaveIP(ip)
	}
	return item.ID, err
}

func startModbusSlave(slave *ModbusSlaveItem) error {
	mc := getConnectionForSlave(slave)
	if mc == nil {
		return logger.Error("Modbus service", "Failed to start modbus slave, no connection object found")
	}

	slave.State = ModbusSlaveStateRunning
	return nil
}

func stopModbusSlave(slave *ModbusSlaveItem) error {
	mc := getConnectionForSlave(slave)
	if mc == nil {
		return logger.Error("Modbus service", "Failed to stop modbus slave, no connection object found")
	}

	slave.State = ModbusSlaveStateStopped
	return nil
}

func InitModbusSlaves() {
	logger.Trace("Modbus service", "Initializing modbus slaves ...")
	abortExistingConnections()

	engineLock.Lock()
	defer engineLock.Unlock()
	db.DB.Find(&slaves)
	db.DB.Find(&datapoints)
	for _, i := range datapoints {
		log.Printf("name: %s, addr: %d, fc: %d, min: %d, max: %d, plcmin: %d, plcmax: %d",
			i.Name, i.ModbusAddress, i.FunctionCode, i.RangeMin, i.RangeMax, i.PlcRangeMin, i.PlcRangeMax)
	}

	buildDatasets(slaves, datapoints)

	logger.Trace("Modbus service", "Done initializing modbus engine!")
}

func StartModbusSlaves(slaves *ModbusSlaveItems) error {
	for _, slave := range slaves.Items {
		startModbusSlave(slave)
	}

	return nil
}

func StopModbusSlaves(slaves *ModbusSlaveItems) error {
	for _, slave := range slaves.Items {
		stopModbusSlave(slave)
	}

	return nil
}

func AddModbusSlave(item *ModbusSlaveItem) error {
	if item.Port == 0 {
		item.Port = 502
	}
	err := db.DB.Create(item).Error
	if err == nil {
		slaves = append(slaves, item)
	}
	return err
}

func AddModbusSlaves(items ModbusSlaveItems) error {
	err := db.DB.Create(items.Items).Error
	if err == nil {
		slaves = append(slaves, items.Items...)
	}
	return err
}

func UpdateModbusSlaves(items ModbusSlaveItems) error {
	err := db.DB.Save(&items.Items).Error
	if err == nil {
		for i, slave := range slaves {
			for _, item := range items.Items {
				if item.ID == slave.ID {
					slaves[i] = item
					break
				}
			}
		}
	}

	return err
}

func DeleteModbusSlaves(items ModbusSlaveItems) error {
	err := db.DB.Delete(items.Items).Error
	if err == nil {
		for si, s := range slaves {
			for _, i := range items.Items {
				if s.ID == i.ID {
					stopModbusSlave(s)
					slaves = append(slaves[:si], slaves[si+1:]...)
					break
				}
			}
		}
	}
	return err
}

func GetModbusConnections() []*modbusConnection {
	return modbusConnections
}

func GetModbusConnection(idx int) *modbusConnection {
	if idx < 0 || idx >= len(modbusConnections) {
		return nil
	}

	return modbusConnections[idx]
}

func GetModbusSlaves() []*ModbusSlaveItem {
	return slaves
}

func RunModbusEngine() {
	log.Printf("Running modbus engine, number of connections: %d", len(modbusConnections))
	for _, mc := range modbusConnections {
		go mc.runSlaveWorker()
	}
}
