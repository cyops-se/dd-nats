package main

import (
	"dd-nats/common/ddnats"
	"dd-nats/ui/dd-ui/db"
	"dd-nats/ui/dd-ui/routes"
	"dd-nats/ui/dd-ui/types"
	"dd-nats/ui/dd-ui/web"
	"flag"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"golang.org/x/sys/windows/svc"
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
var nc *nats.Conn

func main() {
	defer handlePanic()

	svcName := "dd-nats/ui/dd-ui"
	flag.StringVar(&ctx.Cmd, "cmd", "debug", "Windows service command (try 'usage' for more info)")
	flag.StringVar(&ctx.Wdir, "workdir", ".", "Sets the working directory for the process")
	flag.BoolVar(&ctx.Trace, "trace", false, "Prints traces of data to the console")
	flag.BoolVar(&ctx.Version, "v", false, "Prints the commit hash and exits")
	flag.Parse()

	routes.SysInfo.GitVersion = GitVersion
	routes.SysInfo.GitCommit = GitCommit

	if ctx.Version {
		fmt.Printf("dd-nats/ui/dd-ui version %s, commit: %s\n", routes.SysInfo.GitVersion, routes.SysInfo.GitCommit)
		return
	}

	if ctx.Cmd == "install" {
		if err := installService(svcName, "dd-ui from cyops-se"); err != nil {
			log.Fatalf("failed to %s %s: %v", ctx.Cmd, svcName, err)
		}
		return
	} else if ctx.Cmd == "remove" {
		if err := removeService(svcName); err != nil {
			log.Fatalf("failed to %s %s: %v", ctx.Cmd, svcName, err)
		}
		return
	}

	nc, _ = ddnats.Connect(nats.DefaultURL)

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
}

func runEngine() {
	defer handlePanic()

	db.ConnectDatabase(ctx)
	go web.RunWeb(ctx, nc)
}
