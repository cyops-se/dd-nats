package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"dd-nats/common/ddsvc"
	"dd-nats/common/types"

	influxd "github.com/influxdata/influxdb1-client"
)

type InfluxDBEmitter struct {
	Host      string               `json:"host"`
	Port      int                  `json:"port"`
	Database  string               `json:"database"`
	Batchsize int                  `json:"batchsize"`
	client    *influxd.Client      `json:"-" gorm:"-"`
	err       error                `json:"-" gorm:"-"`
	messages  chan types.DataPoint `json:"-" gorm:"-"`
	points    []influxd.Point      `json:"-" gorm:"-"`
	count     uint64               `json:"-" gorm:"-"`
	svc       *ddsvc.DdUsvc        `json:"-" gorm:"-"`
}

func (emitter *InfluxDBEmitter) InitEmitter(svc *ddsvc.DdUsvc) error {
	emitter.svc = svc
	emitter.connectdb()
	emitter.initBatch()
	emitter.messages = make(chan types.DataPoint, 2000)

	go emitter.processMessages()

	return nil
}

func (emitter *InfluxDBEmitter) ProcessDataPointHandler(topic string, responseTopic string, data []byte) error {
	var dp types.DataPoint
	if err := json.Unmarshal(data, &dp); err == nil {
		emitter.ProcessMessage(dp)
	} else {
		emitter.svc.Error("InfluxDB usvc", "Failed to unmarshal process data: %s", err.Error())
	}

	return nil
}

func (emitter *InfluxDBEmitter) ProcessMessage(dp types.DataPoint) {
	if emitter.messages != nil {
		emitter.messages <- dp
	}
}

func (emitter *InfluxDBEmitter) processMessages() {

	for {
		dp := <-emitter.messages

		if emitter.client == nil {
			if emitter.connectdb() != nil {
				continue
			}
		}

		if emitter.appendPoint(&dp) {
			emitter.insertBatch()
			emitter.initBatch()
		}
	}
}

func (emitter *InfluxDBEmitter) connectdb() error {
	host, _ := url.Parse("http://localhost:8086")
	emitter.client, emitter.err = influxd.NewClient(influxd.Config{
		URL: *host,
	})

	if emitter.err == nil {
		emitter.svc.Log("info", "InfluxDB emitter", "Database server connected: localhost")
	} else {
		emitter.svc.Log("info", "InfluxDB emitter", fmt.Sprintf("Database server connect: localhost, failed: %s", emitter.err.Error()))
	}

	return emitter.err
}

func (emitter *InfluxDBEmitter) initBatch() {
	emitter.Batchsize = 5
	emitter.count = 0
	emitter.points = make([]influxd.Point, emitter.Batchsize)
}

func (emitter *InfluxDBEmitter) appendPoint(dp *types.DataPoint) bool {
	if emitter.count < uint64(emitter.Batchsize) {
		emitter.points[emitter.count] = influxd.Point{
			Measurement: "opc",
			Fields: map[string]interface{}{
				dp.Name:              dp.Value,
				dp.Name + "_quality": dp.Quality,
			},
			Time: dp.Time,
		}
		emitter.count++
	}

	return emitter.count >= uint64(emitter.Batchsize)
}

func (emitter *InfluxDBEmitter) insertBatch() error {
	if emitter.count > 0 {
		bps := influxd.BatchPoints{
			Points:          emitter.points,
			Database:        "process",
			RetentionPolicy: "autogen",
		}
		if _, emitter.err = emitter.client.Write(bps); emitter.err != nil {
			emitter.svc.Log("info", "InfluxDB emitter", fmt.Sprintf("Failed to insert new data: %s", emitter.err.Error()))
		}
	}

	return nil
}
