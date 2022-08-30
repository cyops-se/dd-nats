package types

import "time"

type DataPointSample struct {
	Timestamp time.Time `json:"t"`
	Identity  string    `json:"i"`
	Value     float64   `json:"v"`
	Quality   int       `json:"q"`
}

type DataPointSamples []DataPointSample
