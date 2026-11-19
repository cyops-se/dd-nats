package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"time"

	"dd-nats/common/ddsvc"
	"dd-nats/common/types"

	influxd "github.com/influxdata/influxdb1-client"
)

const (
	maxPendingPoints = 100000
	flushInterval    = time.Second
	maxWriteRetries  = 30 // with one retry per second, data is kept for about 30 seconds
)

type InfluxDBEmitter struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	Batchsize int    `json:"batchsize"`

	client   *influxd.Client
	messages chan types.DataPoint
	stop     chan chan struct{}
	points   []influxd.Point
	failures int
	svc      *ddsvc.DdUsvc
}

func (emitter *InfluxDBEmitter) InitEmitter(svc *ddsvc.DdUsvc) error {
	emitter.svc = svc
	if emitter.Batchsize < 1 {
		emitter.Batchsize = 1
	}

	emitter.messages = make(chan types.DataPoint, 2000)
	emitter.stop = make(chan chan struct{})
	emitter.connectdb()

	go emitter.processMessages()
	return nil
}

// Stop flushes pending points and stops the emitter
func (emitter *InfluxDBEmitter) Stop() {
	done := make(chan struct{})
	select {
	case emitter.stop <- done:
		select {
		case <-done:
		case <-time.After(10 * time.Second):
		}
	case <-time.After(time.Second):
	}
}

func (emitter *InfluxDBEmitter) ProcessDataPointHandler(topic string, responseTopic string, data []byte) error {
	var dp types.DataPoint
	if err := json.Unmarshal(data, &dp); err != nil {
		return fmt.Errorf("failed to unmarshal process data: %s", err.Error())
	}

	emitter.ProcessMessage(dp)
	return nil
}

func (emitter *InfluxDBEmitter) ProcessMessage(dp types.DataPoint) {
	if emitter.messages != nil {
		emitter.messages <- dp
	}
}

func (emitter *InfluxDBEmitter) processMessages() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case dp := <-emitter.messages:
			if math.IsNaN(dp.Value) || math.IsInf(dp.Value, 0) {
				continue // not supported by InfluxDB
			}

			emitter.appendPoint(&dp)
			if len(emitter.points) >= emitter.Batchsize {
				emitter.insertBatch()
			}

		case <-ticker.C:
			emitter.insertBatch()

		case done := <-emitter.stop:
			emitter.insertBatch()
			close(done)
			return
		}
	}
}

func (emitter *InfluxDBEmitter) connectdb() error {
	host, err := url.Parse("http://" + net.JoinHostPort(emitter.Host, strconv.Itoa(emitter.Port)))
	if err != nil {
		return emitter.svc.Error("InfluxDB emitter", "Invalid host/port %s:%d: %s", emitter.Host, emitter.Port, err.Error())
	}

	client, err := influxd.NewClient(influxd.Config{URL: *host, Timeout: 30 * time.Second})
	if err != nil {
		return emitter.svc.Error("InfluxDB emitter", "Database server connect: %s, failed: %s", host.String(), err.Error())
	}

	emitter.client = client
	emitter.svc.Info("InfluxDB emitter", "Database server client created: %s", host.String())
	return nil
}

func (emitter *InfluxDBEmitter) appendPoint(dp *types.DataPoint) {
	emitter.points = append(emitter.points, influxd.Point{
		Measurement: "opc",
		Fields: map[string]interface{}{
			dp.Name:              dp.Value,
			dp.Name + "_quality": dp.Quality,
		},
		Time: dp.Time,
	})

	if excess := len(emitter.points) - maxPendingPoints; excess > 0 {
		emitter.points = append([]influxd.Point(nil), emitter.points[excess:]...)
		emitter.svc.Error("InfluxDB emitter", "Buffer full, %d oldest point(s) dropped", excess)
	}
}

func (emitter *InfluxDBEmitter) insertBatch() {
	if len(emitter.points) == 0 {
		return
	}

	if emitter.client == nil && emitter.connectdb() != nil {
		return // keep the points and retry later
	}

	bps := influxd.BatchPoints{
		Points:          emitter.points,
		Database:        emitter.Database,
		RetentionPolicy: "autogen",
	}

	if _, err := emitter.client.Write(bps); err != nil {
		emitter.failures++
		if emitter.failures < maxWriteRetries {
			emitter.svc.Error("InfluxDB emitter", "Failed to insert %d point(s), will retry: %s", len(emitter.points), err.Error())
			return
		}

		// Most likely a problem with the data itself, don't block all other data forever
		emitter.svc.Error("InfluxDB emitter", "Failed to insert %d point(s) %d times, dropped: %s", len(emitter.points), emitter.failures, err.Error())
	}

	emitter.failures = 0
	emitter.points = emitter.points[:0]
}
