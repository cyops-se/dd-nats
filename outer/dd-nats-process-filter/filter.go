package main

import (
	"dd-nats/common/types"
	"encoding/json"
	"log"
	"math"
	"os"
	"time"
)

const (
	FilterTypeNone     = 0
	FilterTypeInterval = 1
	FilterTypeDeadband = 2
)

type allMetaResponse struct {
	types.StatusResponse
	Items types.DataPointMetas `json:"items"`
}

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
	LastTime      time.Time       `json:"-"`
}

var datapoints map[string]*filteredPoint

func processMsgHandler(topic string, responseTopic string, data []byte) error {
	var msg types.DataPointSample
	if err := json.Unmarshal(data, &msg); err == nil {
		for _, dp := range msg.Points {
			fp, ok := datapoints[dp.Name]
			if !ok {
				fp = &filteredPoint{}
				datapoints[dp.Name] = fp
			}

			fp.DataPoint = dp

			svc.Publish("process.actual", dp)
			if fp.FilterType == FilterTypeNone {
				svc.Publish("process.filtered", dp)
			} else if fp.FilterType == FilterTypeInterval {
				if time.Since(fp.PreviousTime) > time.Second*time.Duration(fp.Interval) {
					svc.Publish("process.filtered", dp)
					fp.PreviousTime = time.Now()
					fp.PreviousValue = dp.Value
				}
			} else if fp.FilterType == FilterTypeDeadband {
				fp.Integrator += dp.Value - fp.PreviousValue
				if math.Abs(fp.Integrator) > fp.Deadband*(fp.Max-fp.Min) {
					svc.Publish("process.filtered", dp)
					fp.Integrator = 0
					fp.PreviousValue = dp.Value
				}
			}

			svc.Publish("process.filtermeta", fp)
		}
	} else {
		svc.Error("Process filter", "Failed to unmarshal process data: %s", err.Error())
	}

	return nil
}

func processDataPointHandler(topic string, responseTopic string, data []byte) error {
	var dp types.DataPoint
	if err := json.Unmarshal(data, &dp); err == nil {
		fp, ok := datapoints[dp.Name]
		if !ok {
			fp = &filteredPoint{}
			datapoints[dp.Name] = fp
		}

		fp.DataPoint = dp

		// No more often than once a second regardless
		if time.Since(fp.LastTime) > time.Second*time.Duration(1) {
			svc.Publish("process.actual", dp)
			fp.LastTime = time.Now()

			if fp.FilterType == FilterTypeNone {
				fp.PreviousValue = dp.Value
				svc.Publish("process.filtered", dp)
			} else if fp.FilterType == FilterTypeInterval {
				if time.Since(fp.PreviousTime) > time.Second*time.Duration(fp.Interval) {
					svc.Publish("process.filtered", dp)
					fp.PreviousTime = time.Now()
				}
			} else if fp.FilterType == FilterTypeDeadband {
				fp.Integrator += dp.Value - fp.PreviousValue
				if math.Abs(fp.Integrator) > fp.Deadband*(fp.Max-fp.Min) {
					svc.Publish("process.filtered", dp)
					fp.Integrator = 0
					fp.PreviousValue = dp.Value
				}
			}

			svc.Publish("process.filtermeta", fp)
		}
	} else {
		svc.Error("Process filter", "Failed to unmarshal process data: %s", err.Error())
	}

	return nil
}

func processMetaUpdate(topic string, responseTopic string, data []byte) error {
	syncMetaWithTimescale()
	return nil
}

func saveFilterMeta() error {
	filename := "filterdata.json"
	if content, err := json.Marshal(datapoints); err == nil {
		err := os.WriteFile(filename, content, 0755)
		return err
	} else {
		return err
	}
}

func loadFilterMeta() error {
	filename := "filterdata.json"
	if content, err := os.ReadFile(filename); err != nil {
		return err
	} else {
		if err := json.Unmarshal(content, &datapoints); err != nil {
			return err
		}
	}

	return syncMetaWithTimescale()
}

func syncMetaWithTimescale() error {
	response, err := svc.Request("usvc.timescale.meta.getall", nil)
	if err != nil {
		log.Printf("failed to get meta from timescale: %s", err.Error())
		return err
	}

	var metaitems allMetaResponse
	if err := json.Unmarshal(response, &metaitems); err != nil {
		return err
	}

	for _, item := range metaitems.Items {
		if fp, ok := datapoints[item.Name]; ok {
			fp.Min = item.MinValue
			fp.Max = item.MaxValue
		}
	}

	return nil
}
