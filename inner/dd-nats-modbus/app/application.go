package app

import (
	"dd-nats/common/db"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-modbus/modbus"
	"time"
)

var TRACE bool

func Init(svc *ddsvc.DdUsvc) bool {
	if err := db.ConnectDatabase(*svc.Context, "dd-modbus.db"); err != nil {
		svc.Error("Local database", "Failed to connect to local database, error: %s", err.Error())
		return false
	}

	modbus.TRACE = svc.Context.Trace

	db.ConfigureTypes(db.DB, &types.Log{}, &types.KeyValuePair{})
	db.ConfigureTypes(db.DB, &modbus.ModbusSlaveItem{}, &modbus.ModbusItem{})

	modbus.InitModbusSlaves(svc)

	return true
}

func RunApp(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Modbus microservice running")
	modbus.RunModbusEngine()

	for {
		time.Sleep(1 * time.Second)
	}
}
