package routes

import (
	"github.com/nats-io/nats.go"
)

var lnc *nats.Conn

func RegisterRoutes(nc *nats.Conn) {
	nc.Subscribe("usvc.file-inner.listfolders", listFolders)
	nc.Subscribe("usvc.file-inner.addfolder", addFolder)
	lnc = nc

}
