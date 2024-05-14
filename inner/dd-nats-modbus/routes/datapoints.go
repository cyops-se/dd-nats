package routes

import (
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-modbus/modbus"
	"encoding/json"
)

func registerModbusItemRoutes() {
	usvc.Subscribe("usvc.modbus.items.getall", getAllModbusItems)
	usvc.Subscribe("usvc.modbus.items.add", addModbusItem)
	usvc.Subscribe("usvc.modbus.items.update", updateModbusItem)
	usvc.Subscribe("usvc.modbus.items.delete", deleteModbusItem)
	usvc.Subscribe("usvc.modbus.items.bulkchanges", bulkChangeModbusItems)
}

func getAllModbusItems(topic string, responseTopic string, data []byte) error {
	var response modbus.ModbusItemsResponse
	response.Success = true
	response.Items = modbus.GetModbusDataItems()
	response.Success = false
	return usvc.Publish(responseTopic, response)
}

func addModbusItem(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.AddModbusSlaves(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func updateModbusItem(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Save slaves
		if err = modbus.UpdateModbusSlaves(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func deleteModbusItem(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.DeleteModbusItems(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func bulkChangeModbusItems(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusBulkItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.BulkChangesModbusItems(items.Items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		} else {
			modbus.RestartModbusEngine()
		}
	}

	return usvc.Publish(responseTopic, response)
}
