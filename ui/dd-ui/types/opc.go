package types

import (
	"time"

	"gorm.io/gorm"
)

const (
	GroupStatusNotRunning         = iota
	GroupStatusRunning            = iota
	GroupStatusRunningWithWarning = iota
)

type OPCGroup struct {
	gorm.Model
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Interval     int        `json:"interval"` // Sampling interval in seconds
	Status       int        `json:"status"`   // 0 = stopped, 1 = running, 2 = running with warning
	LastRun      time.Time  `json:"lastrun"`
	Counter      uint       `json:"counter"`
	RunAtStart   bool       `json:"runatstart"`
	LastError    string     `json:"lasterror"`
	ProgID       string     `json:"progid"`
	DiodeProxyID uint       `json:"diodeproxyid"`
	DiodeProxy   DiodeProxy `json:"diodeproxy"`
	DefaultGroup bool       `json:"defaultgroup"`
}

type OPCTag struct {
	gorm.Model
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Integrator          float64  `json:"integrator"`
	IntegratingDeadband float64  `json:"integratingdeadband"`
	GroupID             uint     `json:"groupid"`
	Group               OPCGroup `json:"group"`
}

type TagsInfos struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}
