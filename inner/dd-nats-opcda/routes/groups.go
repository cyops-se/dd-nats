package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/app"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func registerGroupRoutes(svc *ddsvc.DdUsvc) {
	ddnats.Subscribe(svc.RouteName("opc", "groups.getall"), getAllOpcGroups)
	ddnats.Subscribe(svc.RouteName("opc", "groups.getbyid"), getOpcGroupById)
	ddnats.Subscribe(svc.RouteName("opc", "groups.add"), addOpcGroups)
	ddnats.Subscribe(svc.RouteName("opc", "groups.update"), updateOpcGroups)
	ddnats.Subscribe(svc.RouteName("opc", "groups.delete"), deleteOpcGroups)
	ddnats.Subscribe(svc.RouteName("opc", "groups.deleteall"), deleteAllOpcGroups)
	ddnats.Subscribe(svc.RouteName("opc", "groups.start"), startOpcGroup)
	ddnats.Subscribe(svc.RouteName("opc", "groups.stop"), stopOpcGroup)
	// ddnats.Subscribe("usvc.opc.groups.getall", getAllOpcGroups)
	// ddnats.Subscribe("usvc.opc.groups.getbyid", getOpcGroupById)
	// ddnats.Subscribe("usvc.opc.groups.add", addOpcGroups)
	// ddnats.Subscribe("usvc.opc.groups.update", updateOpcGroups)
	// ddnats.Subscribe("usvc.opc.groups.delete", deleteOpcGroups)
	// ddnats.Subscribe("usvc.opc.groups.deleteall", deleteAllOpcGroups)
	// ddnats.Subscribe("usvc.opc.groups.start", startOpcGroup)
	// ddnats.Subscribe("usvc.opc.groups.stop", stopOpcGroup)
}

func getAllOpcGroups(nmsg *nats.Msg) {
	var err error
	var response messages.OpcGroupItemsResponse
	response.Success = true
	if response.Items, err = app.GetGroups(); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}

func getOpcGroupById(nmsg *nats.Msg) {
	var err error
	var response messages.OpcGroupItemResponse
	response.Success = true

	var intmsg types.IntMessage
	if err = json.Unmarshal(nmsg.Data, &intmsg); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if response.Item, err = app.GetGroup(uint(intmsg.Value)); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}

func addOpcGroups(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}
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
	response := types.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		// Clear default flag for all but one
		for n, item := range items.Items {
			if item.DefaultGroup {
				for i := n + 1; i < len(items.Items); i++ {
					items.Items[i].DefaultGroup = false
				}

				if err := db.DB.Exec("update opc_group_items set 'default_group' = false").Error; err != nil {
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
	response := types.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(nmsg.Data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		for _, g := range items.Items {
			app.StopGroup(&g)
		}

		if err = db.DB.Delete(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	ddnats.Respond(nmsg, response)
}

func deleteAllOpcGroups(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}
	var items []app.OpcTagItem
	if err := db.DB.Delete(&items, "1 = 1").Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}
	ddnats.Respond(nmsg, response)
}

func startOpcGroup(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}

	var intmsg types.IntMessage
	var group app.OpcGroupItem
	if err := json.Unmarshal(nmsg.Data, &intmsg); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if group, err = app.GetGroup(uint(intmsg.Value)); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	if err := app.StartGroup(&group); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}
func stopOpcGroup(nmsg *nats.Msg) {
	response := types.StatusResponse{Success: true}

	var intmsg types.IntMessage
	var group app.OpcGroupItem
	if err := json.Unmarshal(nmsg.Data, &intmsg); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if group, err = app.GetGroup(uint(intmsg.Value)); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	if err := app.StopGroup(&group); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}
