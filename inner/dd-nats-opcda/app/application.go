package app

import (
	"dd-nats/common/db"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"time"
)

func Init(svc *ddsvc.DdUsvc) bool {
	if err := db.ConnectDatabase(*svc.Context, "dd-opcda.db"); err != nil {
		logger.Error("Local database", "Failed to connect to local database, error: %s", err.Error())
		return false
	}

	db.ConfigureTypes(db.DB, &types.Log{}, &types.KeyValuePair{})
	db.ConfigureTypes(db.DB, &OpcGroupItem{}, &OpcTagItem{})

	InitGroups(svc)

	return true
}

func RunApp(svc *ddsvc.DdUsvc) {
	logger.Info("Microservices", "OPC DA microservice running")

	for {
		time.Sleep(1 * time.Second)
	}
}
