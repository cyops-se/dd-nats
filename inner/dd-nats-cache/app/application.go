package app

import (
	"dd-nats/common/ddsvc"
	"time"
)

func Init(svc *ddsvc.DdUsvc) bool {

	InitCache(svc)

	return true
}

func RunApp(svc *ddsvc.DdUsvc) {
	svc.Info("Microservices", "Process cache microservice running")

	for {
		time.Sleep(1 * time.Second)
	}
}
