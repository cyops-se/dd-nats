package types

import "time"

// Version 2 message types
type DataPoint struct {
	ID      int         `json:"id"`
	Time    time.Time   `json:"t"`
	Name    string      `json:"n"`
	Value   interface{} `json:"v"`
	Quality int         `json:"q"`
}

type DataMessage struct {
	Version  int         `json:"version"`
	Group    string      `json:"group"`
	Interval int         `json:"interval"`
	Sequence uint64      `json:"sequence"`
	Count    int         `json:"count"`
	Points   []DataPoint `json:"points"`
}

type DataPointMeta struct {
	ID   int    `json:"id"`
	Name string `json:"n"`
}
