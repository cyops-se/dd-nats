package types

import (
	"database/sql"
	"time"

	mqtt "github.com/eclipse/paho.golang/paho"
	"github.com/nats-io/nats.go"
)

type SystemInformation struct {
	GitVersion string `json:"gitversion"`
	GitCommit  string `json:"gitcommit"`
	BuildTime  string `json:"buildtime"`
}

type Model struct {
	ID        uint         `gorm:"primarykey" json:"id"`
	CreatedAt time.Time    `json:"-"`
	UpdatedAt time.Time    `json:"-"`
	DeletedAt sql.NullTime `gorm:"index" json:"-"`
}

type UdpStatistics struct {
	TotalMsg  uint64 `json:"totalmsg"`
	TotalPkts uint64 `json:"totalpkts"`
}

type Heartbeat struct {
	Hostname  string    `json:"hostname"`
	Identity  string    `json:"identity"`
	AppName   string    `json:"appname"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

type PlainMessage struct {
	Message string `json:"message"`
}

type IntMessage struct {
	Value int `json:"value"`
}

const (
	ConnectionTypeUnknown = 0
	ConnectionTypeNATS    = 1
	ConnectionTypeMQTT    = 2
)

type Connection struct {
	Error      error
	ConType    int
	NatsCon    *nats.Conn
	MqttClient *mqtt.Client
	MqttCon    *mqtt.Connack
}
