package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type allFilteredPointsResponse struct {
	ddsvc.StatusResponse
	Items []*filteredPoint `json:"items"`
}

func registerFilterRoutes() {
	ddnats.Subscribe("usvc.process.filter.getall", getAllFilteredPoints)
	ddnats.Subscribe("usvc.process.filter.setfilter", setFilterPoint)
}

func getAllFilteredPoints(nmsg *nats.Msg) {
	var response allFilteredPointsResponse
	response.Success = true

	var items []*filteredPoint
	for _, v := range datapoints {
		items = append(items, v)
	}

	response.Items = items

	ddnats.Respond(nmsg, response)
}

func setFilterPoint(nmsg *nats.Msg) {
	var response ddsvc.StatusResponse
	response.Success = true

	var item filteredPoint
	if err := json.Unmarshal(nmsg.Data, &item); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
		log.Println("request body:", string(nmsg.Data), ", error:", err.Error())
	} else {
		datapoints[item.DataPoint.Name] = &item
	}

	ddnats.Respond(nmsg, response)
}
