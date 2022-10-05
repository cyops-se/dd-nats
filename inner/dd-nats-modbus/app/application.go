package app

import (
	"dd-nats/common/db"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-modbus/modbus"
	"time"
)

var TRACE bool

func Init(ctx *types.Context) bool {
	if err := db.ConnectDatabase(*ctx, "dd-modbus.db"); err != nil {
		logger.Error("Local database", "Failed to connect to local database, error: %s", err.Error())
		return false
	}

	modbus.TRACE = ctx.Trace

	db.ConfigureTypes(db.DB, &types.Log{}, &types.KeyValuePair{})
	db.ConfigureTypes(db.DB, &modbus.ModbusSlaveItem{}, &modbus.ModbusItem{})

	modbus.InitModbusSlaves()

	return true
}

func RunApp(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "Modbus microservice running")
	modbus.RunModbusEngine()

	for {
		time.Sleep(1 * time.Second)
	}
}
