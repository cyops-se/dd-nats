package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/inner/dd-nats-opcda/data"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

func registerGroupRoutes() {
	ddnats.Subscribe("usvc.opc.groups.getall", getAllOpcGroups)
	ddnats.Subscribe("usvc.opc.groups.add", addOpcGroups)
	ddnats.Subscribe("usvc.opc.groups.update", updateOpcGroups)
	ddnats.Subscribe("usvc.opc.groups.delete", deleteOpcGroups)
	ddnats.Subscribe("usvc.opc.groups.deleteall", deleteAllOpcGroups)
}

func getAllOpcGroups(nmsg *nats.Msg) {
	var response messages.OpcGroupItemResponse
	response.Success = true
	if err := db.DB.Find(&response.Items).Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}

func addOpcGroups(nmsg *nats.Msg) {
	log.Println("add payload:", string(nmsg.Data))
	response := messages.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Create(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func updateOpcGroups(nmsg *nats.Msg) {
	log.Println("update payload:", string(nmsg.Data))
	response := messages.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Clear default flag for all but one
		for n, item := range items.Items {
			if item.Default {
				for i := n + 1; i < len(items.Items); i++ {
					items.Items[i].Default = false
				}

				if err := db.DB.Exec("update opc_group_items set 'default' = false").Error; err != nil {
					logger.Error("updateOpcGroups error", "failed to reset default group flag, error: %s", err.Error())
				}
				break
			}
		}

		// Save groups
		if err = db.DB.Save(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteOpcGroups(nmsg *nats.Msg) {
	response := messages.StatusResponse{Success: true}
	var items messages.Groups
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

func deleteAllOpcGroups(nmsg *nats.Msg) {
	response := messages.StatusResponse{Success: true}
	var dbitems []data.OpcTagItem
	if err := db.DB.Delete(&dbitems, "1 = 1").Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}
	ddnats.Respond(nmsg, response)
}
