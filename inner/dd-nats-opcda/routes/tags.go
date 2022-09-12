package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/inner/dd-nats-opcda/data"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm/clause"
)

func registerTagRoutes() {
	ddnats.Subscribe("usvc.opc.tags.getall", getAllOpcTags)
	ddnats.Subscribe("usvc.opc.tags.add", addOpcTags)
	ddnats.Subscribe("usvc.opc.tags.update", updateOpcTags)
	ddnats.Subscribe("usvc.opc.tags.delete", deleteOpcTags)
	ddnats.Subscribe("usvc.opc.tags.deleteall", deleteAllOpcTags)
}

func getAllOpcTags(nmsg *nats.Msg) {
	var response messages.OpcTagItemResponse
	response.Success = true
	if err := db.DB.Preload(clause.Associations).Find(&response.Items).Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}

func addOpcTags(nmsg *nats.Msg) {
	response := messages.StatusResponse{Success: true}
	var item messages.Tag
	if err := json.Unmarshal(nmsg.Data, &item); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		dbitem := data.OpcTagItem{}
		dbitem.Name = item.Tag
		if err = db.DB.Create(&dbitem).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func updateOpcTags(nmsg *nats.Msg) {
	response := messages.StatusResponse{Success: true}
	var item data.OpcTagItem
	if err := json.Unmarshal(nmsg.Data, &item); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Save(&item).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteOpcTags(nmsg *nats.Msg) {
	response := messages.StatusResponse{Success: false, StatusMessage: "Method not yet implemented: deleteOpcTags"}
	ddnats.Respond(nmsg, response)
}

func deleteAllOpcTags(nmsg *nats.Msg) {
	response := messages.StatusResponse{Success: true}
	var dbitems []data.OpcTagItem
	if err := db.DB.Delete(&dbitems, "1 = 1").Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}
	ddnats.Respond(nmsg, response)
}
