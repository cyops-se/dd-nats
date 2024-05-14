package routes

import (
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/messages"
	"log"
	"os"
	"path"
)

func registerFolderRoutes() {
	log.Println("Registering folder routes")
	usvc.Subscribe("usvc.filetransfer.listfolders", listFolders)
}

func listFolders(topic string, responseTopic string, data []byte) error {
	var response messages.FolderInfo
	response.Success = true

	cwd, _ := os.Getwd()
	ctx := app.Context()
	response.NewDir = path.Join(cwd, "outgoing", ctx.NewDir)
	response.ProcessingDir = path.Join(cwd, "outgoing", ctx.ProcessingDir)
	response.FailDir = path.Join(cwd, "outgoing", ctx.FailDir)
	response.DoneDir = path.Join(cwd, "outgoing", ctx.DoneDir)
	return usvc.Publish(responseTopic, response)
}
