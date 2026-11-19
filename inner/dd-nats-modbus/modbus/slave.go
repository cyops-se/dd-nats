package modbus

import (
	"dd-nats/common/db"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	mb "github.com/simonvetter/modbus"
)

// Max number of registers that can be read in one request according to the Modbus specification
const maxRegistersPerRead = 125

type modbusDataset struct {
	start uint16
	count uint16
	fc    uint16
	items []*ModbusItem
}

type modbusConnection struct {
	client    *mb.ModbusClient
	Connected bool             `json:"connected"` // Open() successful
	Slave     *ModbusSlaveItem `json:"slave"`
	Datasets  []*modbusDataset `json:"datasets"`
	ErrStr    string           `json:"error"`
	Err       error            `json:"-"`
	stop      chan struct{}
}

type ItemByAddress []*ModbusItem

var usvc *ddsvc.DdUsvc

// lock protects slaves, datapoints, modbusConnections and the runtime state
// fields of the slaves (State, LastRun, LastError, ErrorMsg, Counter)
var lock sync.Mutex
var modbusConnections []*modbusConnection
var slaves []*ModbusSlaveItem
var datapoints []*ModbusItem

var workers sync.WaitGroup
var restartLock sync.Mutex

var TRACE bool

func (a ItemByAddress) Len() int      { return len(a) }
func (a ItemByAddress) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ItemByAddress) Less(i, j int) bool {
	if a[i].FunctionCode != a[j].FunctionCode {
		return a[i].FunctionCode < a[j].FunctionCode
	}
	return a[i].ModbusAddress < a[j].ModbusAddress
}

// transposeAddress converts the conventional 3xxxx/4xxxx register numbers to
// a protocol address
func transposeAddress(item *ModbusItem) uint16 {
	if item.FunctionCode == 3 && item.ModbusAddress >= 40000 {
		item.AdaptedAddress = item.ModbusAddress - 40000
	} else if item.FunctionCode == 4 && item.ModbusAddress >= 30000 {
		item.AdaptedAddress = item.ModbusAddress - 30000
	} else {
		item.AdaptedAddress = item.ModbusAddress
	}

	return uint16(item.AdaptedAddress)
}

// scaleValue converts a raw register value to engineering units and returns
// the value and OPC quality. If no PLC range is configured, the raw value is used.
func scaleValue(item *ModbusItem, rawvalue uint16) (float64, int) {
	const qualityGood = 192 // OPC quality GOOD (non-specific)
	const qualityBad = 0    // OPC quality BAD (non-specific)

	if item.PlcRangeMin == 0 && item.PlcRangeMax == 0 {
		return float64(rawvalue), qualityGood
	}

	if item.PlcRangeMax <= item.PlcRangeMin {
		return float64(rawvalue), qualityBad // invalid configuration, avoid division by zero
	}

	factor := (float64(rawvalue) - float64(item.PlcRangeMin)) / (float64(item.PlcRangeMax) - float64(item.PlcRangeMin))
	value := (float64(item.RangeMax)-float64(item.RangeMin))*factor + float64(item.RangeMin)

	if rawvalue < item.PlcRangeMin || rawvalue > item.PlcRangeMax {
		return value, qualityBad
	}

	return value, qualityGood
}

// buildDatasets groups the (sorted) items into datasets of consecutive
// registers with the same function code, each at most maxRegistersPerRead long
func buildDatasets(slave *ModbusSlaveItem, items []*ModbusItem) []*modbusDataset {
	sort.Sort(ItemByAddress(items))

	var datasets []*modbusDataset
	var ds *modbusDataset

	for n, item := range items {
		newset := ds == nil ||
			item.FunctionCode != items[n-1].FunctionCode ||
			item.ModbusAddress != items[n-1].ModbusAddress+1 ||
			ds.count >= maxRegistersPerRead

		if newset {
			ds = &modbusDataset{
				fc:    uint16(item.FunctionCode),
				start: uint16(int(transposeAddress(item)) + slave.Offset),
			}
			datasets = append(datasets, ds)
		}

		ds.items = append(ds.items, item)
		ds.count++
	}

	return datasets
}

// Must be called with lock held
func buildConnections() {
	usvc.Trace("Modbus TCP", "Building datasets ...")

	for _, slave := range slaves {
		usvc.Trace("Modbus TCP", " ... for slave: %s, ip: %s, port: %d, offset: %d", slave.Name, slave.IPAddress, slave.Port, slave.Offset)

		items := findItemsForSlave(slave, datapoints)
		if len(items) == 0 {
			usvc.Error("Modbus TCP", "No items available for modbus slave %s", slave.IPAddress)
			continue
		}

		mc := &modbusConnection{Slave: slave, Datasets: buildDatasets(slave, items), stop: make(chan struct{})}

		if TRACE {
			for dno, ds := range mc.Datasets {
				usvc.Trace("Modbus TCP", "ds: %d, fc: %d, start: %d, count: %d, offset: %d, slave ip: %s",
					dno, ds.fc, ds.start, ds.count, slave.Offset, slave.IPAddress)
				for i, item := range ds.items {
					usvc.Trace("Modbus TCP", "item: %d, name: %s, address: %d, slave ip: %s", i, item.Name, item.ModbusAddress, slave.IPAddress)
				}
			}
		}

		slave.State = ModbusSlaveStateRunning
		modbusConnections = append(modbusConnections, mc)
	}

	usvc.Trace("Modbus TCP", "Done building datasets for %d Modbus slaves!", len(slaves))
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

func (mc *modbusConnection) setError(err error) {
	lock.Lock()
	defer lock.Unlock()

	mc.Err = err
	mc.ErrStr = err.Error()
	mc.Slave.LastError = time.Now().UTC().Format("2006-01-02 15:04:05")
	mc.Slave.ErrorMsg = err.Error()
}

func (mc *modbusConnection) checkConnection() error {
	if mc.client == nil {
		client, err := mb.NewClient(&mb.ClientConfiguration{
			URL:     fmt.Sprintf("tcp://%s:%d", mc.Slave.IPAddress, mc.Slave.Port),
			Timeout: 1 * time.Second,
		})

		if err != nil {
			mc.setError(err)
			return err
		}

		mc.client = client
	}

	if !mc.Connected {
		if err := mc.client.Open(); err != nil {
			mc.setError(err)
			return err
		}

		mc.Connected = true
		usvc.Trace("Modbus TCP", "Modbus client connection successful for slave: %s", mc.Slave.IPAddress)

		lock.Lock()
		mc.Err = nil
		mc.ErrStr = ""
		mc.Slave.ErrorMsg = ""
		lock.Unlock()
	}

	return nil
}

func (mc *modbusConnection) closeConnection() {
	if mc != nil && mc.client != nil {
		mc.client.Close()
		mc.client = nil
	}
	mc.Connected = false
}

func (mc *modbusConnection) readDataset(dsno int, ds *modbusDataset) error {
	var rawvalues []uint16
	var err error

	switch ds.fc {
	case 3:
		rawvalues, err = mc.client.ReadRegisters(ds.start, ds.count, mb.HOLDING_REGISTER)
	case 4:
		rawvalues, err = mc.client.ReadRegisters(ds.start, ds.count, mb.INPUT_REGISTER)
	default:
		return fmt.Errorf("modbus function code not supported: %d", ds.fc)
	}

	if err != nil {
		return fmt.Errorf("failure to read registers, start: %d, count: %d, error: %s", ds.start, ds.count, err.Error())
	}

	now := time.Now().UTC()
	for n, rawvalue := range rawvalues {
		if n >= len(ds.items) {
			break
		}

		item := ds.items[n]
		value, quality := scaleValue(item, rawvalue)
		dp := types.DataPoint{Time: now, Name: item.Name, Value: value, Quality: quality}
		usvc.Publish("process.actual", dp)

		usvc.Trace("Modbus TCP", "ds: %d, name: %s, address: %d, raw value: %v, value: %f, quality: %d, slave ip: %s",
			dsno, item.Name, int(ds.start)+n, rawvalue, value, quality, mc.Slave.IPAddress)
	}

	lock.Lock()
	mc.Slave.Counter += uint64(len(rawvalues))
	lock.Unlock()

	return nil
}

// Function code 3 = Holding registers, 4 = Input registers
// NOTE! This is a very limited implementation right now that only support 16 bit uint values
func (mc *modbusConnection) runSlaveWorker() {
	defer workers.Done()
	defer mc.closeConnection()

	interval := time.Duration(mc.Slave.Interval) * time.Second
	if interval <= 0 {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		lock.Lock()
		stopped := mc.Slave.State == ModbusSlaveStateStopped
		lock.Unlock()

		if stopped {
			mc.closeConnection()
		} else if mc.checkConnection() == nil {
			for dsno, ds := range mc.Datasets {
				if err := mc.readDataset(dsno, ds); err != nil {
					usvc.Error("Modbus TCP", "Slave %s: %s", mc.Slave.IPAddress, err.Error())
					mc.setError(err)
					mc.closeConnection()
					break
				}
			}

			lock.Lock()
			mc.Slave.LastRun = time.Now().UTC().Format("2006-01-02 15:04:05")
			lock.Unlock()
		}

		select {
		case <-mc.stop:
			return
		case <-ticker.C:
		}
	}
}

// Stops all workers and waits for them to exit
func abortExistingConnections() {
	lock.Lock()
	for _, mc := range modbusConnections {
		close(mc.stop)
	}
	modbusConnections = nil
	lock.Unlock()

	workers.Wait()
}

func getSlaveByID(id uint) *ModbusSlaveItem {
	for _, slave := range slaves {
		if slave.ID == id {
			return slave
		}
	}

	return nil
}

func checkSlaveIP(posteditem *BulkChangeModbusItem) (uint, error) {
	if posteditem.ModbusSlaveID > 0 {
		return posteditem.ModbusSlaveID, nil
	}

	ip := strings.TrimSpace(posteditem.IPAddress)
	if ip == "" {
		return 0, usvc.Error("Modbus TCP", "Failed to check empty modbus slave IP for item %s", posteditem.Name)
	}

	var item ModbusSlaveItem
	if err := db.DB.First(&item, "ip_address = ?", ip).Error; err == nil {
		return item.ID, nil
	}

	item = ModbusSlaveItem{IPAddress: ip, Name: ip}
	if err := AddModbusSlave(&item); err != nil {
		return 0, usvc.Error("Modbus TCP", "Failed to create modbus slave %s: %s", ip, err.Error())
	}

	return item.ID, nil
}

func setSlaveState(requested *ModbusSlaveItem, state int) error {
	lock.Lock()
	defer lock.Unlock()

	slave := getSlaveByID(requested.ID)
	if slave == nil {
		return usvc.Error("Modbus TCP", "Failed to change state of modbus slave, no slave with id %d found", requested.ID)
	}

	// Runtime state only, all slaves are started when the engine (re)starts
	slave.State = state
	return nil
}

func InitModbusSlaves(svc *ddsvc.DdUsvc) {
	TRACE = svc.Context.Trace
	usvc = svc

	usvc.Trace("Modbus TCP", "Initializing modbus slaves ...")
	abortExistingConnections()

	lock.Lock()
	defer lock.Unlock()

	slaves = nil
	datapoints = nil
	if err := db.DB.Find(&slaves).Error; err != nil {
		usvc.Error("Modbus TCP", "Failed to load modbus slaves: %s", err.Error())
	}

	if err := db.DB.Find(&datapoints).Error; err != nil {
		usvc.Error("Modbus TCP", "Failed to load modbus items: %s", err.Error())
	}

	if svc.Context.Trace {
		for _, i := range datapoints {
			usvc.Trace("Modbus TCP", "name: %s, addr: %d, fc: %d, min: %d, max: %d, plcmin: %d, plcmax: %d",
				i.Name, i.ModbusAddress, i.FunctionCode, i.RangeMin, i.RangeMax, i.PlcRangeMin, i.PlcRangeMax)
		}
	}

	buildConnections()

	usvc.Trace("Modbus TCP", "Done initializing modbus engine!")
}

func StartModbusSlaves(items *ModbusSlaveItems) error {
	var lasterr error
	for _, slave := range items.Items {
		if err := setSlaveState(slave, ModbusSlaveStateRunning); err != nil {
			lasterr = err
		}
	}

	return lasterr
}

func StopModbusSlaves(items *ModbusSlaveItems) error {
	var lasterr error
	for _, slave := range items.Items {
		if err := setSlaveState(slave, ModbusSlaveStateStopped); err != nil {
			lasterr = err
		}
	}

	return lasterr
}

func AddModbusSlave(item *ModbusSlaveItem) error {
	if item.Port == 0 {
		item.Port = 502
	}

	if err := db.DB.Create(item).Error; err != nil {
		return err
	}

	lock.Lock()
	slaves = append(slaves, item)
	lock.Unlock()
	return nil
}

func AddModbusSlaves(items ModbusSlaveItems) error {
	for _, item := range items.Items {
		if err := AddModbusSlave(item); err != nil {
			return err
		}
	}

	return nil
}

func UpdateModbusSlaves(items ModbusSlaveItems) error {
	if len(items.Items) == 0 {
		return nil
	}

	if err := db.DB.Save(&items.Items).Error; err != nil {
		return err
	}

	lock.Lock()
	defer lock.Unlock()
	for i, slave := range slaves {
		for _, item := range items.Items {
			if item.ID == slave.ID {
				slaves[i] = item
				break
			}
		}
	}

	return nil
}

func DeleteModbusSlaves(items ModbusSlaveItems) error {
	if len(items.Items) == 0 {
		return nil
	}

	if err := db.DB.Delete(items.Items).Error; err != nil {
		return err
	}

	deleted := make(map[uint]bool, len(items.Items))
	for _, i := range items.Items {
		deleted[i.ID] = true
	}

	lock.Lock()
	defer lock.Unlock()

	remaining := slaves[:0]
	for _, s := range slaves {
		if deleted[s.ID] {
			s.State = ModbusSlaveStateStopped
		} else {
			remaining = append(remaining, s)
		}
	}
	slaves = remaining

	return nil
}

// GetModbusSlaves returns a snapshot of all slaves including their runtime state
func GetModbusSlaves() []*ModbusSlaveItem {
	lock.Lock()
	defer lock.Unlock()

	result := make([]*ModbusSlaveItem, 0, len(slaves))
	for _, s := range slaves {
		c := *s
		result = append(result, &c)
	}

	return result
}

func RunModbusEngine() {
	lock.Lock()
	defer lock.Unlock()

	usvc.Info("Modbus TCP", "Running modbus engine, number of connections: %d", len(modbusConnections))
	for _, mc := range modbusConnections {
		workers.Add(1)
		go mc.runSlaveWorker()
	}
}

// RestartModbusEngine reloads the configuration and restarts all workers.
// Restarts are serialized.
func RestartModbusEngine() {
	restartLock.Lock()
	defer restartLock.Unlock()

	usvc.Info("Modbus TCP", "Restarting modbus engine")
	InitModbusSlaves(usvc)
	RunModbusEngine()
}

// RestartModbusEngineAsync restarts the engine in the background so message
// handlers are not blocked while workers are stopped
func RestartModbusEngineAsync() {
	go RestartModbusEngine()
}
