package app

import (
	"dd-nats/common/types"
	"time"
)

const (
	GroupStateUnknown            = 0
	GroupStateStopped            = 1
	GroupStateRunning            = 2
	GroupStateRunningWithWarning = 3
)

type OpcGroupItem struct {
	types.Model
	Name         string    `json:"name"`
	ProgID       string    `json:"progid"`
	Interval     int       `json:"interval"`
	RunAtStart   bool      `json:"runatstart"`
	DefaultGroup bool      `json:"defaultgroup"`
	State        int       `json:"state"`
	Counter      uint64    `json:"counter"`
	LastRun      time.Time `json:"lastrun"`
}

type OpcTagItem struct {
	types.Model
	Name    string       `json:"name"`
	Group   OpcGroupItem `json:"group"`
	GroupID int          `json:"groupid"`
}
