package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/messages"
	"log"
	"os"
	"path"

	"github.com/nats-io/nats.go"
)

func registerFolderRoutes() {
	log.Println("Registering folder routes")
	ddnats.Subscribe("usvc.filetransfer.listfolders", listFolders)
}

func listFolders(msg *nats.Msg) {
	var response messages.FolderInfo
	response.Success = true

	cwd, _ := os.Getwd()
	ctx := app.Context()
	response.NewDir = path.Join(cwd, "outgoing", ctx.NewDir)
	response.ProcessingDir = path.Join(cwd, "outgoing", ctx.ProcessingDir)
	response.FailDir = path.Join(cwd, "outgoing", ctx.FailDir)
	response.DoneDir = path.Join(cwd, "outgoing", ctx.DoneDir)
	ddnats.Respond(msg, response)
}
