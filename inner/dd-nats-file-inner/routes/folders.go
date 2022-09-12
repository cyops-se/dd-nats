package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/common/types"

	"github.com/nats-io/nats.go"
)

func registerFolderRoutes() {
	ddnats.Subscribe("usvc.file-inner.listfolders", listFolders)
	ddnats.Subscribe("usvc.file-inner.addfolder", addFolder)
}

func listFolders(msg *nats.Msg) {
	logger.Trace("listfolders received", "%s", string(msg.Data))
	var response []*types.FolderInfo
	response = append(response, &types.FolderInfo{Name: "kalle", Subject: "sk"})
	response = append(response, &types.FolderInfo{Name: "anka", Subject: "sa"})
	ddnats.Respond(msg, response)
}

func addFolder(msg *nats.Msg) {
	logger.Trace("registerfolder received", "%s", string(msg.Data))
	ddnats.PublishError("not yet implemented: %s", "[testing variardic]")
}
