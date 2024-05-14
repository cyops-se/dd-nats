package routes

import "dd-nats/common/ddsvc"

var usvc *ddsvc.DdUsvc

func RegisterRoutes(svc *ddsvc.DdUsvc) {
	usvc = svc
	registerModbusItemRoutes()
	registerModbusSlaveRoutes()
}
