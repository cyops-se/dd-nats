package data

import (
	"dd-nats/common/db"
	"dd-nats/common/logger"
	"dd-nats/common/types"
)

const (
	GroupStateUnknown = 0
	GroupStateStopped = 1
	GroupStateRunning = 2
)

type OpcGroupItem struct {
	types.Model
	Name       string `json:"name"`
	ProgID     string `json:"progid"`
	Interval   int    `json:"interval"`
	RunAtStart bool   `json:"runatstart"`
	Default    bool   `json:"defaultgroup"`
	State      int    `json:"state"`
}

type OpcTagItem struct {
	types.Model
	types.DataPointInfo
	Group   OpcGroupItem `json:"group"`
	GroupID int          `json:"groupid"`
}

func InitLocalDatabase(ctx *types.Context) bool {
	if err := db.ConnectDatabase(*ctx, "dd-opcda.db"); err != nil {
		logger.Error("Local database", "Failed to connect to local database, error: %s", err.Error())
		return false
	}

	db.ConfigureTypes(db.DB, &types.Log{})
	db.ConfigureTypes(db.DB, &OpcGroupItem{}, &OpcTagItem{})
	return true
}
