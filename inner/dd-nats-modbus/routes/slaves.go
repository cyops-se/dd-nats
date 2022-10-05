package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/inner/dd-nats-modbus/modbus"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func RegisterModbusSlaveRoutes() {
	ddnats.Subscribe("usvc.modbus.slaves.getall", getAllModbusSlaves)
	ddnats.Subscribe("usvc.modbus.slaves.add", addModbusSlaves)
	ddnats.Subscribe("usvc.modbus.slaves.update", updateModbusSlaves)
	ddnats.Subscribe("usvc.modbus.slaves.delete", deleteModbusSlaves)
	ddnats.Subscribe("usvc.modbus.slaves.start", startModbusSlave)
	ddnats.Subscribe("usvc.modbus.slaves.stop", stopModbusSlave)
}

func getAllModbusSlaves(nmsg *nats.Msg) {
	var response modbus.ModbusSlaveItemsResponse
	response.Success = true
	response.Items = modbus.GetModbusSlaves()
	response.Success = false
	ddnats.Respond(nmsg, response)
}

func addModbusSlaves(nmsg *nats.Msg) {
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

func updateModbusSlaves(nmsg *nats.Msg) {
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

func deleteModbusSlaves(nmsg *nats.Msg) {
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

func startModbusSlave(nmsg *nats.Msg) {
	response := ddsvc.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Save slaves
		if err = modbus.StartModbusSlaves(&items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func stopModbusSlave(nmsg *nats.Msg) {
	response := ddsvc.StatusResponse{Success: true}
	var items modbus.ModbusSlaveItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Save slaves
		if err = modbus.StopModbusSlaves(&items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}
