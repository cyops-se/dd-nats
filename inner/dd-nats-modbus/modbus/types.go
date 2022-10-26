package modbus

import "dd-nats/common/types"

// Common

const (
	ModbusSlaveStateUnknown            = 0
	ModbusSlaveStateStopped            = 1
	ModbusSlaveStateRunning            = 2
	ModbusSlaveStateRunningWithWarning = 3
)

type Tag struct {
	Tag string `json:"tag"`
}

type Tags struct {
	Items []Tag `json:"items"`
}

type ModbusItems struct {
	Items []*ModbusItem `json:"items"`
}

type ModbusSlaveItems struct {
	Items []*ModbusSlaveItem `json:"items"`
}

type ModbusBulkItems struct {
	Items []*BulkChangeModbusItem `json:"items"`
}

// Response

type ModbusItemsResponse struct {
	types.StatusResponse
	Items []*ModbusItem `json:"items"`
}

type ModbusSlaveItemResponse struct {
	types.StatusResponse
	Item ModbusSlaveItem `json:"item"`
}

type ModbusSlaveItemsResponse struct {
	types.StatusResponse
	Items []*ModbusSlaveItem `json:"items"`
}
