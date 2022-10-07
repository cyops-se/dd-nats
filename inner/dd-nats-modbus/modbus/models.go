package modbus

import (
	"dd-nats/common/types"
)

type ModbusSlaveItem struct {
	types.Model
	Name         string `json:"name"`
	IPAddress    string `json:"ip"`
	Port         int    `json:"port"`
	Offset       int    `json:"offset"`
	Interval     int    `json:"interval"`
	RunAtStart   bool   `json:"runatstart"`
	DefaultGroup bool   `json:"defaultgroup"`
	State        int    `json:"state"`
	Counter      uint64 `json:"counter"`
	LastRun      string `json:"lastrun"`
	LastError    string `json:"lasterror"`
	ErrorMsg     string `json:"errormsg"`
}

type ModbusItem struct {
	types.Model
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	EngUnit        string          `json:"engunit"`
	FunctionCode   uint            `json:"functioncode"`
	ModbusAddress  uint            `json:"modbusaddress"`
	AdaptedAddress uint            `json:"adaptedaddress"`
	ByteOrder      string          `json:"byteorder"`
	DataType       string          `json:"datatype"`
	DataLength     uint            `json:"datalength"`
	RangeMin       uint16          `json:"rangemin"`
	RangeMax       uint16          `json:"rangemax"`
	PlcRangeMin    uint16          `json:"plcrangemin"`
	PlcRangeMax    uint16          `json:"plcrangemax"`
	ModbusSlave    ModbusSlaveItem `json:"modbusslave"`
	ModbusSlaveID  uint            `json:"modbusslaveid"`
}

type BulkChangeModbusItem struct {
	types.Model
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IPAddress     string          `json:"ipaddress"`
	EngUnit       string          `json:"engunit"`
	FunctionCode  uint            `json:"functioncode"`
	ModbusAddress uint            `json:"modbusaddress"`
	ByteOrder     string          `json:"byteorder"`
	DataType      string          `json:"datatype"`
	DataLength    uint            `json:"datalength"`
	RangeMin      uint16          `json:"rangemin"`
	RangeMax      uint16          `json:"rangemax"`
	PlcRangeMin   uint16          `json:"plcrangemin"`
	PlcRangeMax   uint16          `json:"plcrangemax"`
	ModbusSlave   ModbusSlaveItem `json:"modbusslave"`
	ModbusSlaveID uint            `json:"modbusslaveid"`
}
