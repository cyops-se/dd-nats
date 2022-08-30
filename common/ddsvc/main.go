package main

import (
	"dd-opcda/db"
	"dd-opcda/engine"
	"dd-opcda/routes"
	"dd-opcda/types"
	"dd-opcda/web"
	"flag"
	"fmt"
	"log"

	"golang.org/x/sys/windows/svc"
)

// type DataPoint struct {
// 	Time    time.Time   `json:"t"`
// 	Name    string      `json:"n"`
// 	Value   interface{} `json:"v"`
// 	Quality int         `json:"q"`
// }

// type DataMessage struct {
// 	Counter uint64      `json:"counter"`
// 	Count   int         `json:"count"`
// 	Points  []DataPoint `json:"points"`
// }

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
	defer handlePanic()

	svcName := "dd-opcda"
	flag.StringVar(&ctx.Cmd, "cmd", "debug", "Windows service command (try 'usage' for more info)")
	flag.StringVar(&ctx.Wdir, "workdir", ".", "Sets the working directory for the process")
	flag.BoolVar(&ctx.Trace, "trace", false, "Prints traces of OCP data to the console")
	flag.BoolVar(&ctx.Version, "v", false, "Prints the commit hash and exits")
	flag.Parse()

	routes.SysInfo.GitVersion = GitVersion
	routes.SysInfo.GitCommit = GitCommit

	if ctx.Version {
		fmt.Printf("dd-opcda version %s, commit: %s\n", routes.SysInfo.GitVersion, routes.SysInfo.GitCommit)
		return
	}

	if ctx.Cmd == "install" {
		if err := installService(svcName, "dd-opcda from cyops-se"); err != nil {
			log.Fatalf("failed to %s %s: %v", ctx.Cmd, svcName, err)
		}
		return
	} else if ctx.Cmd == "remove" {
		if err := removeService(svcName); err != nil {
			log.Fatalf("failed to %s %s: %v", ctx.Cmd, svcName, err)
		}
		return
	}

	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}
	if inService {
		runService(svcName, false)
		return
	}

	// runEngine()
	runService(svcName, true)
	engine.CloseCache()
}

func runEngine() {
	defer handlePanic()

	db.ConnectDatabase(ctx)
	engine.InitGroups()
	engine.InitServers()
	engine.InitCache()
	engine.InitFileTransfer(ctx)
	go web.RunWeb(ctx)
}
