package routes

import (
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/messages"
)

func registerManifestRoutes() {
	usvc.Subscribe("usvc.filetransfer.getmanifest", getManifest)
}

func getManifest(topic string, responseTopic string, data []byte) error {
	var response messages.Manifest
	response.Success = true
	response.Manifest = app.GetManifest()

	return usvc.Publish(responseTopic, response)
}
