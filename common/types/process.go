package types

import "time"

type DataPointInfo struct {
	Name string `json:"name"`
}

type DataPointValue struct {
	Name    string  `json:"n"`
	Value   float64 `json:"v"`
	Quality int     `json:"q"`
}

type DataPointValues []DataPointValue

type DataPointSample struct {
	Timestamp time.Time       `json:"time"`
	Values    DataPointValues `json:"values"`
}

type DataPointSamples []DataPointSample
