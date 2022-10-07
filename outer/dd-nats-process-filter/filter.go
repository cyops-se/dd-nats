package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"encoding/json"
	"math"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	FilterTypeNone     = 0
	FilterTypeInterval = 1
	FilterTypeDeadband = 2
)

// Three types of filters is provided
// 0. No filtering, forward as is
// 1. Interval, ensure the point is not forwarded more frequent than the specified interval (in seconds)
// 2. Deadband, forward when accumulated difference exceed specified threshold (as a percentage of max-min)
type filteredPoint struct {
	DataPoint     types.DataPoint `json:"datapoint"`
	FilterType    int             `json:"filtertype"`
	PreviousTime  time.Time       `json:"previoustime"`
	Interval      int             `json:"interval"`
	PreviousValue float64         `json:"previousvalue"`
	Integrator    float64         `json:"integrator"`
	Deadband      float64         `json:"deadband"` // Percentage of max-min
	Min           float64         `json:"min"`      // Retrieved from Meta service (for example dd-nats-timescale-meta)
	Max           float64         `json:"max"`      // Retrieved from Meta service (for example dd-nats-timescale-meta)
}

var datapoints map[string]*filteredPoint

func processMsgHandler(nmsg *nats.Msg) {
	var msg types.DataPointSample
	if err := json.Unmarshal(nmsg.Data, &msg); err == nil {
		for _, dp := range msg.Points {
			fp, ok := datapoints[dp.Name]
			if !ok {
				fp = &filteredPoint{}
				datapoints[dp.Name] = fp
			}

			fp.DataPoint = dp

			ddnats.Publish("process.actual", dp)
			if fp.FilterType == FilterTypeNone {
				ddnats.Publish("process.filtered", dp)
			} else if fp.FilterType == FilterTypeInterval {
				if time.Since(fp.PreviousTime) > time.Second*time.Duration(fp.Interval) {
					ddnats.Publish("process.filtered", dp)
					fp.PreviousTime = time.Now()
					fp.PreviousValue = dp.Value
				}
			} else if fp.FilterType == FilterTypeDeadband {
				fp.Integrator += dp.Value - fp.PreviousValue
				if math.Abs(fp.Integrator) > fp.Deadband*(fp.Max-fp.Min) {
					ddnats.Publish("process.filtered", dp)
					fp.Integrator = 0
					fp.PreviousValue = dp.Value
				}
			}
			ddnats.Publish("process.filtermeta", fp)

		}
	} else {
		logger.Error("Timescale server", "Failed to unmarshal process data: %s", err.Error())
	}
}

func processDataPointHandler(nmsg *nats.Msg) {
	var dp types.DataPoint
	if err := json.Unmarshal(nmsg.Data, &dp); err == nil {
		fp, ok := datapoints[dp.Name]
		if !ok {
			fp = &filteredPoint{}
			datapoints[dp.Name] = fp
		}

		fp.DataPoint = dp

		ddnats.Publish("process.actual", dp)
		if fp.FilterType == FilterTypeNone {
			ddnats.Publish("process.filtered", dp)
		} else if fp.FilterType == FilterTypeInterval {
			if time.Since(fp.PreviousTime) > time.Second*time.Duration(fp.Interval) {
				ddnats.Publish("process.filtered", dp)
				fp.PreviousTime = time.Now()
				fp.PreviousValue = dp.Value
			}
		} else if fp.FilterType == FilterTypeDeadband {
			fp.Integrator += dp.Value - fp.PreviousValue
			if math.Abs(fp.Integrator) > fp.Deadband*(fp.Max-fp.Min) {
				ddnats.Publish("process.filtered", dp)
				fp.Integrator = 0
				fp.PreviousValue = dp.Value
			}
		}
		ddnats.Publish("process.filtermeta", fp)
	} else {
		logger.Error("Timescale server", "Failed to unmarshal process data: %s", err.Error())
	}
}
