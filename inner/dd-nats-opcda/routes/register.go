package routes

import "dd-nats/common/ddsvc"

func RegisterRoutes(svc *ddsvc.DdUsvc) {
	registerSystemRoutes(svc)
	registerGroupRoutes(svc)
	registerTagRoutes(svc)
	registerOpcRoutes(svc)
}
