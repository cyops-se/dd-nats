// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package ddsvc

import (
	"dd-nats/common/types"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

type myservice struct {
	usvc   *DdUsvc
	engine func(*DdUsvc)
}

var SysInfo types.SystemInformation

func handlePanic() {
	if r := recover(); r != nil {
		// ddsvc.Error("Windows service error", "Panic, recovery: %#v", r)
		log.Printf("Windows server panic, recovery: %#v", r)
		return
	}
}

func processArgs(svcName string) *types.Context {
	ctx := &types.Context{}
	flag.StringVar(&ctx.Cmd, "cmd", "debug", "Windows service command (try 'usage' for more info)")
	flag.StringVar(&ctx.Wdir, "workdir", ".", "Sets the working directory for the process")
	flag.BoolVar(&ctx.Trace, "trace", false, "Prints traces from the application to the console")
	flag.BoolVar(&ctx.Version, "v", false, "Prints the commit hash and exits")
	flag.StringVar(&ctx.Name, "name", svcName, "Sets the name of the service")
	flag.StringVar(&ctx.Url, "url", "nats://localhost:4222", "URL to NATS service")
	flag.IntVar(&ctx.Port, "port", 3000, "Port for HTTP user interface, if supported by service")
	flag.StringVar(&ctx.Id, "id", "default", "Service instance identity. Important when running multiple instances of the same service")
	flag.Parse()

	SysInfo.GitVersion = GitVersion
	SysInfo.GitCommit = GitCommit
	SysInfo.BuildTime = BuildTime

	if ctx.Version {
		fmt.Printf("%s version %s, commit: %s, build time: %s\n", svcName, SysInfo.GitVersion, SysInfo.GitCommit, SysInfo.BuildTime)
		return nil
	}

	if ctx.Cmd == "install" {
		if err := installService(svcName, fmt.Sprintf("%s from cyops-se", svcName)); err != nil {
			log.Fatalf("failed to %s %s: %v", ctx.Cmd, svcName, err)
		}
		return nil
	} else if ctx.Cmd == "remove" {
		if err := removeService(svcName); err != nil {
			log.Fatalf("failed to %s %s: %v", ctx.Cmd, svcName, err)
		}
		return nil
	}

	return ctx
}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	defer handlePanic()

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	m.usvc.Info("Service engine", "Starting engine %s", m.usvc.Name)
	go m.engine(m.usvc)

	m.usvc.Info("Service engine", "Entering service control loop")

loop:
	for {
		select {
		case <-tick:
			// beep()
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				// golang.org/x/sys/windows/svc.TestExample is verifying this output.
				testOutput := strings.Join(args, "-")
				testOutput += fmt.Sprintf("-%d", c.Context)
				m.usvc.Info("Service engine", testOutput)
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
			default:
				m.usvc.Error("Service engine", "Unexpected control request #%v", c)
			}
		}
	}
	m.usvc.Info("Service engine", "Exiting service control loop")
	changes <- svc.Status{State: svc.StopPending}
	return
}

func RunService(usvc *DdUsvc, engine func(*DdUsvc)) {
	var err error
	usvc.Info("Service engine", "Starting %s service", usvc.Name)
	run := svc.Run

	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %s", err.Error())
	}
	if !inService {
		run = debug.Run
	}

	err = run(usvc.Name, &myservice{engine: engine, usvc: usvc})
	if err != nil {
		usvc.Error("Service engine", "%s service failed: %s", usvc.Name, err.Error())
		return
	}
	usvc.Info("Service engine", "%s service stopped", usvc.Name)
}
