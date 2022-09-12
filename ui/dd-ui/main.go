package main

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"dd-nats/ui/dd-ui/web"
	"log"

	"github.com/nats-io/nats.go"
)

type ConfigTagEntry struct {
	Name string `json:"name"`
}

type Config struct {
	Tags []ConfigTagEntry `json:"tags"`
}

var ctx types.Context
var GitVersion string
var GitCommit string

func main() {
	svcName := "dd-ui"
	_, err := ddnats.Connect(nats.DefaultURL)
	if err != nil {
		log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
		return
	}

	ctx := ddsvc.ProcessArgs(svcName)
	if ctx == nil {
		return
	}

	db.ConnectDatabase(*ctx, "dd-ui.db")
	db.ConfigureTypes(db.DB, &types.User{}, &types.Log{})

	go ddnats.SendHeartbeat(svcName)
	ddsvc.RunService(svcName, runEngine)

	log.Printf("Exiting ...")
}

func runEngine() {
	go web.RunWeb(ctx)
}
