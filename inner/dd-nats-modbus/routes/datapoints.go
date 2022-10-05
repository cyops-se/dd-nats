package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-modbus/modbus"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func RegisterModbusItemRoutes() {
	ddnats.Subscribe("usvc.modbus.items.getall", getAllModbusItems)
	ddnats.Subscribe("usvc.modbus.items.add", addModbusItem)
	ddnats.Subscribe("usvc.modbus.items.update", updateModbusItem)
	ddnats.Subscribe("usvc.modbus.items.delete", deleteModbusItem)
	ddnats.Subscribe("usvc.modbus.items.bulkchanges", bulkChangeModbusItems)
}

func getAllModbusItems(nmsg *nats.Msg) {
	var response modbus.ModbusItemsResponse
	response.Success = true
	response.Items = modbus.GetModbusDataItems()
	response.Success = false
	ddnats.Respond(nmsg, response)
}

func addModbusItem(nmsg *nats.Msg) {
	response := ddsvc.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.AddModbusSlaves(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func updateModbusItem(nmsg *nats.Msg) {
	response := ddsvc.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Save slaves
		if err = modbus.UpdateModbusSlaves(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteModbusItem(nmsg *nats.Msg) {
	response := ddsvc.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.DeleteModbusSlaves(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func bulkChangeModbusItems(nmsg *nats.Msg) {
	response := ddsvc.StatusResponse{Success: true}
	var items modbus.ModbusBulkItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.BulkChangesModbusItems(items.Items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}
	ddnats.Respond(nmsg, response)
}
