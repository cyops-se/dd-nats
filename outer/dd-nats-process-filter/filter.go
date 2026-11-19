package main

import (
	"dd-nats/common/types"
	"encoding/json"
	"math"
	"os"
	"sync"
	"time"
)

const (
	FilterTypeNone     = 0
	FilterTypeInterval = 1
	FilterTypeDeadband = 2
)

const filterDataFile = "filterdata.json"

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

// datapoints is accessed from several message handlers and the periodic
// submission task concurrently, always hold datapointsLock
var datapoints = make(map[string]*filteredPoint)
var datapointsLock sync.Mutex

type publication struct {
	topic string
	data  interface{}
}

// filter applies the configured filter to a data point and returns the
// messages to publish. Must be called with datapointsLock held.
func filter(fp *filteredPoint, dp types.DataPoint, now time.Time) []publication {
	var out []publication

	out = append(out, publication{"process.actual", dp})
	switch fp.FilterType {
	case FilterTypeNone:
		fp.PreviousValue = dp.Value
		out = append(out, publication{"process.filtered", dp})

	case FilterTypeInterval:
		if now.Sub(fp.PreviousTime) > time.Second*time.Duration(fp.Interval) {
			out = append(out, publication{"process.filtered", dp})
			fp.PreviousTime = now
			fp.PreviousValue = dp.Value
		}

	case FilterTypeDeadband:
		fp.Integrator += dp.Value - fp.PreviousValue
		if math.Abs(fp.Integrator) > fp.Deadband*(fp.Max-fp.Min) {
			out = append(out, publication{"process.filtered", dp})
			fp.Integrator = 0
			fp.PreviousValue = dp.Value
		}
	}

	out = append(out, publication{"process.filtermeta", *fp})
	return out
}

func processDataPointHandler(topic string, responseTopic string, data []byte) error {
	var dp types.DataPoint
	if err := json.Unmarshal(data, &dp); err != nil {
		svc.Error("Process filter", "Failed to unmarshal process data: %s", err.Error())
		return nil
	}

	now := time.Now()

	datapointsLock.Lock()
	fp, ok := datapoints[dp.Name]
	if !ok {
		fp = &filteredPoint{}
		datapoints[dp.Name] = fp
	}

	fp.DataPoint = dp

	// No more often than once a second regardless
	var out []publication
	if now.Sub(fp.LastTime) > time.Second {
		fp.LastTime = now
		out = filter(fp, dp, now)
	}
	datapointsLock.Unlock()

	for _, p := range out {
		svc.Publish(p.topic, p.data)
	}

	return nil
}

// The accumulated deadband in this filter process is dependent on periodic
// collection of samples. This background task will periodically check all tags
// and submit the last seen value with quality 68 for those that haven't been seen
// in the last 10 secs. This function has been moved here from the sampler on the inside.
func processPeriodSubmissionTask() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var resubmit []types.DataPoint

		datapointsLock.Lock()
		for _, fp := range datapoints {
			if now.Sub(fp.LastTime) > 10*time.Second {
				dp := fp.DataPoint
				dp.Quality = 68 // Uncertain [Last usable] tag.Quality;
				dp.Time = now.UTC()
				fp.LastTime = now
				resubmit = append(resubmit, dp)
			}
		}
		datapointsLock.Unlock()

		// Fed back through the regular input topic, so all consumers see a consistent stream
		for _, dp := range resubmit {
			svc.Publish(inputTopic, dp)
		}
	}
}

func processMetaUpdate(topic string, responseTopic string, data []byte) error {
	return syncMetaWithTimescale()
}

// Must be called with datapointsLock held
func saveFilterMeta() error {
	content, err := json.Marshal(datapoints)
	if err != nil {
		return err
	}

	tmpname := filterDataFile + ".tmp"
	if err = os.WriteFile(tmpname, content, 0644); err != nil {
		return err
	}

	return os.Rename(tmpname, filterDataFile)
}

func loadFilterMeta() error {
	content, err := os.ReadFile(filterDataFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err == nil {
		loaded := make(map[string]*filteredPoint)
		if err := json.Unmarshal(content, &loaded); err != nil {
			return err
		}

		datapointsLock.Lock()
		for k, v := range loaded {
			if v != nil {
				datapoints[k] = v
			}
		}
		datapointsLock.Unlock()
	}

	return syncMetaWithTimescale()
}

func syncMetaWithTimescale() error {
	response, err := svc.Request("usvc.timescale.meta.getall", nil)
	if err != nil {
		return svc.Error("Process filter", "Failed to get meta from timescale: %s", err.Error())
	}

	var metaitems allMetaResponse
	if err := json.Unmarshal(response, &metaitems); err != nil {
		return err
	}

	datapointsLock.Lock()
	defer datapointsLock.Unlock()

	for _, item := range metaitems.Items {
		if fp, ok := datapoints[item.Name]; ok {
			fp.Min = item.MinValue
			fp.Max = item.MaxValue
		}
	}

	return nil
}
