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
	response.Success = false
	return usvc.Publish(responseTopic, response)
}

func addModbusSlaves(topic string, responseTopic string, data []byte) error {
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

func updateModbusSlaves(topic string, responseTopic string, data []byte) error {
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

func deleteModbusSlaves(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err := modbus.DeleteModbusSlaves(items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func startModbusSlave(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Save slaves
		if err = modbus.StartModbusSlaves(&items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func stopModbusSlave(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Save slaves
		if err = modbus.StopModbusSlaves(&items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}
