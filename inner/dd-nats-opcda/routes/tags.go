package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/app"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"
	"log"

	"gorm.io/gorm/clause"
)

func registerTagRoutes() {
	usvc.Subscribe(usvc.RouteName("opc", "tags.getall"), getAllOpcTags)
	usvc.Subscribe(usvc.RouteName("opc", "tags.add"), addOpcTags)
	usvc.Subscribe(usvc.RouteName("opc", "tags.update"), updateOpcTags)
	usvc.Subscribe(usvc.RouteName("opc", "tags.delete"), deleteOpcTags)
	usvc.Subscribe(usvc.RouteName("opc", "tags.deletebyname"), deleteOpcTagByName)
	usvc.Subscribe(usvc.RouteName("opc", "tags.deleteall"), deleteAllOpcTags)
}

func getAllOpcTags(topic string, responseTopic string, data []byte) error {
	var response messages.OpcTagItemResponse
	response.Success = true
	if err := db.DB.Preload(clause.Associations).Find(&response.Items).Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}

	return usvc.Publish(responseTopic, response)
}

func addOpcTags(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items messages.Tags
	if err := json.Unmarshal(data, &items); err != nil {
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

	return usvc.Publish(responseTopic, response)
}

func updateOpcTags(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items messages.OpcItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Save(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func deleteOpcTags(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items messages.OpcItems
	if err := json.Unmarshal(data, &items); err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	} else {
		if err = db.DB.Delete(&items.Items).Error; err != nil {
			response.Success = false
			response.StatusMessage = err.Error()
		}
	}

	return usvc.Publish(responseTopic, response)
}

func deleteOpcTagByName(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var items messages.Tags
	if err := json.Unmarshal(data, &items); err != nil {
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

	return usvc.Publish(responseTopic, response)
}

func deleteAllOpcTags(topic string, responseTopic string, data []byte) error {
	response := types.StatusResponse{Success: true}
	var dbitems []app.OpcTagItem
	if err := db.DB.Delete(&dbitems, "1 = 1").Error; err != nil {
		response.Success = false
		response.StatusMessage = err.Error()
	}
	return usvc.Publish(responseTopic, response)
}
