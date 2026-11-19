package routes

import (
	"dd-nats/inner/dd-nats-file-inner/app"
	"dd-nats/inner/dd-nats-file-inner/messages"
)

func registerFolderRoutes() {
	usvc.Subscribe("usvc.filetransfer.listfolders", listFolders)
}

func listFolders(topic string, responseTopic string, data []byte) error {
	var response messages.FolderInfo

	ctx := app.Context()
	if ctx == nil {
		response.StatusMessage = "file transfer engine not initialized"
		return usvc.Publish(responseTopic, response)
	}

	response.Success = true
	response.NewDir = ctx.Dir(ctx.NewDir)
	response.ProcessingDir = ctx.Dir(ctx.ProcessingDir)
	response.FailDir = ctx.Dir(ctx.FailDir)
	response.DoneDir = ctx.Dir(ctx.DoneDir)
	return usvc.Publish(responseTopic, response)
}
