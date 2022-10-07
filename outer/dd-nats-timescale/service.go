package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type updateMetaRequest struct {
	Items types.DataPointMetas `json:"items"`
}

type allMetaResponse struct {
	ddsvc.StatusResponse
	Items types.DataPointMetas `json:"items"`
}

func registerRoutes() {
	ddnats.Subscribe("usvc.timescale.meta.getall", getAllMeta)
	ddnats.Subscribe("usvc.timescale.meta.updateall", updateAllMeta)
}

func getAllMeta(nmsg *nats.Msg) {
	var err error
	var response allMetaResponse
	response.Success = true

	if response.Items, err = getAllMetaFromDatabase(); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}

func updateAllMeta(nmsg *nats.Msg) {
	response := &ddsvc.StatusResponse{Success: false, StatusMessage: "Not yet implemented"}

	var request updateMetaRequest
	if err := json.Unmarshal(nmsg.Data, &request); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = updateAllMetaInDatabase(request.Items); err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}
