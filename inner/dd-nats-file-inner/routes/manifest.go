package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/messages"
	"log"

	"github.com/nats-io/nats.go"
)

func registerManifestRoutes() {
	log.Println("Registering manifest routes")
	ddnats.Subscribe("usvc.filetransfer.getmanifest", getManifest)
}

func getManifest(msg *nats.Msg) {
	var response messages.Manifest
	response.Success = true
	response.Manifest = *app.Manifest

	ddnats.Respond(msg, response)
}
