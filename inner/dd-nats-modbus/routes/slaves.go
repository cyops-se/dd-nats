package routes

import (
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-modbus/modbus"
	"encoding/json"
)

func registerModbusSlaveRoutes() {
	usvc.Subscribe("usvc.modbus.slaves.getall", getAllModbusSlaves)
	usvc.Subscribe("usvc.modbus.slaves.add", addModbusSlaves)
	usvc.Subscribe("usvc.modbus.slaves.update", updateModbusSlaves)
	usvc.Subscribe("usvc.modbus.slaves.delete", deleteModbusSlaves)
	usvc.Subscribe("usvc.modbus.slaves.start", startModbusSlave)
	usvc.Subscribe("usvc.modbus.slaves.stop", stopModbusSlave)
}

func getAllModbusSlaves(topic string, responseTopic string, data []byte) error {
	var response modbus.ModbusSlaveItemsResponse
	response.Success = true
	response.Items = modbus.GetModbusSlaves()
	return usvc.Publish(responseTopic, response)
}

// handleSlaves unmarshals a list of slaves, applies the operation and
// optionally restarts the engine (in the background) if the operation succeeded
func handleSlaves(responseTopic string, data []byte, restart bool, operation func(*modbus.ModbusSlaveItems) error) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if err := operation(&items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if restart {
		modbus.RestartModbusEngineAsync()
	}

	return usvc.Publish(responseTopic, response)
}

func addModbusSlaves(topic string, responseTopic string, data []byte) error {
	return handleSlaves(responseTopic, data, true, func(items *modbus.ModbusSlaveItems) error {
		return modbus.AddModbusSlaves(*items)
	})
}

func updateModbusSlaves(topic string, responseTopic string, data []byte) error {
	return handleSlaves(responseTopic, data, true, func(items *modbus.ModbusSlaveItems) error {
		return modbus.UpdateModbusSlaves(*items)
	})
}

func deleteModbusSlaves(topic string, responseTopic string, data []byte) error {
	return handleSlaves(responseTopic, data, true, func(items *modbus.ModbusSlaveItems) error {
		return modbus.DeleteModbusSlaves(*items)
	})
}

func startModbusSlave(topic string, responseTopic string, data []byte) error {
	return handleSlaves(responseTopic, data, false, modbus.StartModbusSlaves)
}

func stopModbusSlave(topic string, responseTopic string, data []byte) error {
	return handleSlaves(responseTopic, data, false, modbus.StopModbusSlaves)
}
