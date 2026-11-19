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
	return usvc.Publish(responseTopic, response)
}

// handleItems unmarshals a list of items, applies the operation and restarts
// the engine (in the background) if the operation succeeded
func handleItems(responseTopic string, data []byte, operation func(modbus.ModbusItems) error) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if err := operation(items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		modbus.RestartModbusEngineAsync()
	}

	return usvc.Publish(responseTopic, response)
}

func addModbusItem(topic string, responseTopic string, data []byte) error {
	return handleItems(responseTopic, data, modbus.AddModbusDataItems)
}

func updateModbusItem(topic string, responseTopic string, data []byte) error {
	return handleItems(responseTopic, data, modbus.UpdateModbusItems)
}

func deleteModbusItem(topic string, responseTopic string, data []byte) error {
	return handleItems(responseTopic, data, modbus.DeleteModbusItems)
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
		}

		// Restart even if some items failed, the others have been saved
		modbus.RestartModbusEngineAsync()
	}

	return usvc.Publish(responseTopic, response)
}
