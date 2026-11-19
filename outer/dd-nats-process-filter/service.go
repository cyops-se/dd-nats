package main

import (
	"dd-nats/common/types"
	"encoding/json"
)

type allFilteredPoints struct {
	Items []*filteredPoint `json:"items"`
}

type allFilteredPointsResponse struct {
	types.StatusResponse
	Items []filteredPoint `json:"items"`
}

func registerFilterRoutes() {
	svc.Subscribe("usvc.process.filter.getall", getAllFilteredPoints)
	svc.Subscribe("usvc.process.filter.setfilter", setFilterPoint)
}

func getAllFilteredPoints(topic string, responseTopic string, data []byte) error {
	var response allFilteredPointsResponse
	response.Success = true

	// Copy the values while holding the lock, they are modified concurrently
	datapointsLock.Lock()
	response.Items = make([]filteredPoint, 0, len(datapoints))
	for _, v := range datapoints {
		response.Items = append(response.Items, *v)
	}
	datapointsLock.Unlock()

	return svc.Publish(responseTopic, response)
}

func setFilterPoint(topic string, responseTopic string, data []byte) error {
	var response types.StatusResponse
	response.Success = true

	var items allFilteredPoints
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
		return svc.Publish(responseTopic, response)
	}

	datapointsLock.Lock()
	for _, item := range items.Items {
		if item == nil || item.DataPoint.Name == "" {
			continue
		}

		// Keep the runtime state of existing points
		if existing, ok := datapoints[item.DataPoint.Name]; ok {
			item.LastTime = existing.LastTime
		}

		datapoints[item.DataPoint.Name] = item
	}

	if err := saveFilterMeta(); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}
	datapointsLock.Unlock()

	return svc.Publish(responseTopic, response)
}
