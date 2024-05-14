package main

import (
	"dd-nats/common/types"
	"encoding/json"
)

type updateMetaRequest struct {
	Items types.DataPointMetas `json:"items"`
}

type allMetaResponse struct {
	types.StatusResponse
	Items types.DataPointMetas `json:"items"`
}

func registerRoutes() {
	svc.Subscribe("usvc.timescale.meta.getall", getAllMeta)
	svc.Subscribe("usvc.timescale.meta.updateall", updateAllMeta)
	svc.Subscribe("usvc.timescale.meta.delete", deleteMeta)
}

func getAllMeta(topic string, responseTopic string, data []byte) error {
	var err error
	var response allMetaResponse
	response.Success = true

	if response.Items, err = getAllMetaFromDatabase(); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	return svc.Publish(responseTopic, response)
}

func updateAllMeta(topic string, responseTopic string, data []byte) error {
	response := &types.StatusResponse{Success: true}

	var request updateMetaRequest
	if err := json.Unmarshal(data, &request); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = updateAllMetaInDatabase(request.Items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return svc.Publish(responseTopic, response)
}

func deleteMeta(topic string, responseTopic string, data []byte) error {
	response := &types.StatusResponse{Success: true}

	var request updateMetaRequest
	if err := json.Unmarshal(data, &request); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = deleteMetaInDatabase(request.Items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return svc.Publish(responseTopic, response)
}
