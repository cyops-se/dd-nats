package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/app"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"
	"log"
)

func registerGroupRoutes() {
	usvc.Subscribe(usvc.RouteName("opc", "groups.getall"), getAllOpcGroups)
	usvc.Subscribe(usvc.RouteName("opc", "groups.getbyid"), getOpcGroupById)
	usvc.Subscribe(usvc.RouteName("opc", "groups.add"), addOpcGroups)
	usvc.Subscribe(usvc.RouteName("opc", "groups.update"), updateOpcGroups)
	usvc.Subscribe(usvc.RouteName("opc", "groups.delete"), deleteOpcGroups)
	usvc.Subscribe(usvc.RouteName("opc", "groups.deleteall"), deleteAllOpcGroups)
	usvc.Subscribe(usvc.RouteName("opc", "groups.start"), startOpcGroup)
	usvc.Subscribe(usvc.RouteName("opc", "groups.stop"), stopOpcGroup)
}

func getAllOpcGroups(topic string, responseTopic string, data []byte) error {
	var err error
	var response messages.OpcGroupItemsResponse
	response.Success = true
	if response.Items, err = app.GetGroups(); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	return usvc.Publish(responseTopic, response)
}

func getOpcGroupById(topic string, responseTopic string, data []byte) error {
	var err error
	var response messages.OpcGroupItemResponse
	response.Success = true

	var intmsg types.IntMessage
	if err = json.Unmarshal(data, &intmsg); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else if response.Item, err = app.GetGroup(uint(intmsg.Value)); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	return usvc.Publish(responseTopic, response)
}

func addOpcGroups(topic string, responseTopic string, data []byte) error {
	log.Printf("Add group, request received: %s", string(data))
	response := types.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Create(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	log.Printf("Add group, response: %v", response)
	return usvc.Publish(responseTopic, response)
}

func updateOpcGroups(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(data, &items); err != nil {
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
					usvc.Error("updateOpcGroups error", "failed to reset default group flag, error: %s", err.Error())
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

	return usvc.Publish(responseTopic, response)
}

func deleteOpcGroups(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items messages.Groups
	if err := json.Unmarshal(data, &items); err != nil {
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

	return usvc.Publish(responseTopic, response)
}

func deleteAllOpcGroups(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items []app.OpcTagItem
	if err := db.DB.Delete(&items, "1 = 1").Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	return usvc.Publish(responseTopic, response)
}

func startOpcGroup(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}

	var intmsg types.IntMessage
	var group app.OpcGroupItem
	if err := json.Unmarshal(data, &intmsg); err != nil {
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

	return usvc.Publish(responseTopic, response)
}
func stopOpcGroup(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}

	var intmsg types.IntMessage
	var group app.OpcGroupItem
	if err := json.Unmarshal(data, &intmsg); err != nil {
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

	return usvc.Publish(responseTopic, response)
}
