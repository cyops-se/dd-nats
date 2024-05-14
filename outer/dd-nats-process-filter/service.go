package main

import (
	"dd-nats/common/types"
	"encoding/json"
	"log"
)

type allFilteredPoints struct {
	Items []*filteredPoint `json:"items"`
}

type allFilteredPointsResponse struct {
	types.StatusResponse
	Items []*filteredPoint `json:"items"`
}

func registerFilterRoutes() {
	svc.Subscribe("usvc.process.filter.getall", getAllFilteredPoints)
	svc.Subscribe("usvc.process.filter.setfilter", setFilterPoint)
}

func getAllFilteredPoints(topic string, responseTopic string, data []byte) error {
	var response allFilteredPointsResponse
	response.Success = true

	var items []*filteredPoint
	for _, v := range datapoints {
		items = append(items, v)
	}

	response.Items = items

	return svc.Publish(responseTopic, response)
}

func setFilterPoint(topic string, responseTopic string, data []byte) error {
	var response types.StatusResponse
	response.Success = true

	var items allFilteredPoints
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
		log.Println("request body:", string(data), ", error:", err.Error())
	} else {
		for _, item := range items.Items {
			datapoints[item.DataPoint.Name] = item
			if err = saveFilterMeta(); err != nil {
				response.Success = false
				response.StatusMessage = err.Error()
			}
		}
	}

	return svc.Publish(responseTopic, response)
}
