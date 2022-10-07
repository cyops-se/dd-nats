package types

import "time"

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

type DataPointMeta struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	EngUnit     string  `json:"engunit"`
	Type        string  `json:"type"`
	MinValue    float64 `json:"min"`
	MaxValue    float64 `json:"max"`
	// The following are filter specific attributes and should be handled there
	// Quantity            string  `json:"quantity"`
	// FilterType          int     `json:"filtertype"` // 0 = pass thru, 1 = interval, 2 = integrating deadband, 3 = disabled
	// Interval            int     `json:"interval"`
	// IntegratingDeadband float64 `json:"integratingdeadband"`
}

type DataPointMetas []DataPointMeta
