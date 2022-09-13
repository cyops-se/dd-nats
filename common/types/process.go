package types

import "time"

type DataPointInfo struct {
	Name string `json:"name"`
}

type DataPoint struct {
	Time    time.Time `json:"t"`
	Name    string    `json:"n"`
	Value   float64   `json:"v"`
	Quality int       `json:"q"`
}

type DataPoints []DataPoint

type DataPointSample struct {
	Version  int        `json:"version"`
	Sequence uint64     `json:"sequence"`
	Group    string     `json:"group"`
	Points   DataPoints `json:"points"`
}

type DataPointSamples []DataPointSample
