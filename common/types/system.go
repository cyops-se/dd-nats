package types

import (
	"database/sql"
	"time"
)

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
