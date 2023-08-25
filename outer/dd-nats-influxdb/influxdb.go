package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"dd-nats/common/logger"
	"dd-nats/common/types"

	influxd "github.com/influxdata/influxdb1-client"

	"github.com/nats-io/nats.go"
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
}

func (emitter *InfluxDBEmitter) InitEmitter() error {
	emitter.connectdb()
	emitter.initBatch()
	emitter.messages = make(chan types.DataPoint, 2000)

	go emitter.processMessages()

	return nil
}

func (emitter *InfluxDBEmitter) ProcessDataPointHandler(nmsg *nats.Msg) {
	var dp types.DataPoint
	if err := json.Unmarshal(nmsg.Data, &dp); err == nil {
		emitter.ProcessMessage(dp)
	} else {
		logger.Error("InfluxDB usvc", "Failed to unmarshal process data: %s", err.Error())
	}
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
		logger.Log("info", "InfluxDB emitter", "Database server connected: localhost")
	} else {
		logger.Log("info", "InfluxDB emitter", fmt.Sprintf("Database server connect: localhost, failed: %s", emitter.err.Error()))
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
			logger.Log("info", "InfluxDB emitter", fmt.Sprintf("Failed to insert new data: %s", emitter.err.Error()))
		}
	}

	return nil
}
