package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/app"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm/clause"
)

func registerTagRoutes(svc *ddsvc.DdUsvc) {
	ddnats.Subscribe(svc.RouteName("opc", "tags.getall"), getAllOpcTags)
	ddnats.Subscribe(svc.RouteName("opc", "tags.add"), addOpcTags)
	ddnats.Subscribe(svc.RouteName("opc", "tags.update"), updateOpcTags)
	ddnats.Subscribe(svc.RouteName("opc", "tags.delete"), deleteOpcTags)
	ddnats.Subscribe(svc.RouteName("opc", "tags.deletebyname"), deleteOpcTagByName)
	ddnats.Subscribe(svc.RouteName("opc", "tags.deleteall"), deleteAllOpcTags)
	// ddnats.Subscribe("usvc.opc.tags.getall", getAllOpcTags)
	// ddnats.Subscribe("usvc.opc.tags.add", addOpcTags)
	// ddnats.Subscribe("usvc.opc.tags.update", updateOpcTags)
	// ddnats.Subscribe("usvc.opc.tags.delete", deleteOpcTags)
	// ddnats.Subscribe("usvc.opc.tags.deletebyname", deleteOpcTagByName)
	// ddnats.Subscribe("usvc.opc.tags.deleteall", deleteAllOpcTags)
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
	response := types.StatusResponse{Success: true}
	var items messages.Tags
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		var groupid uint
		group, err := app.GetDefaultGroup()
		if err == nil && group != nil {
			groupid = group.ID
		} else {
			log.Println("Failed to find default group, err:", err.Error())
		}

		for _, item := range items.Items {
			dbitem := app.OpcTagItem{GroupID: int(groupid)}
			dbitem.Name = item.Tag
			if err = db.DB.Create(&dbitem).Error; err != nil {
				response.Success = false
				response.StatusMessage = err.Error()
			}
		}
	}

	ddnats.Respond(nmsg, response)
}

func updateOpcTags(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}
	var items messages.OpcItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Save(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteOpcTags(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}
	var items messages.OpcItems
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Delete(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteOpcTagByName(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}
	var items messages.Tags
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		for _, item := range items.Items {
			if err = db.DB.Delete(&app.OpcTagItem{}, "name = ?", item.Tag).Error; err != nil {
				response.Success = false
				response.StatusMessage = err.Error()
			}
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteAllOpcTags(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}
	var dbitems []app.OpcTagItem
	if err := db.DB.Delete(&dbitems, "1 = 1").Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}
	ddnats.Respond(nmsg, response)
}
