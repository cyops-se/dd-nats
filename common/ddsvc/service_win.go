// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package ddsvc

import (
	"dd-nats/common/logger"
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

func handlePanic() {
	if r := recover(); r != nil {
		// logger.Error("Windows service error", "Panic, recovery: %#v", r)
		log.Printf("Windows server panic, recovery: %#v", r)
		return
	}
}

func reportError(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	logger.Error("Windows service error", msg)
}

func reportInfo(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	logger.Trace("Windows service info", msg)
}

func processArgs(svcName string) *types.Context {
	ctx := &types.Context{}
	flag.StringVar(&ctx.Cmd, "cmd", "debug", "Windows service command (try 'usage' for more info)")
	flag.StringVar(&ctx.Wdir, "workdir", ".", "Sets the working directory for the process")
	flag.BoolVar(&ctx.Trace, "trace", false, "Prints traces from the application to the console")
	flag.BoolVar(&ctx.Version, "v", false, "Prints the commit hash and exits")
	flag.StringVar(&ctx.Name, "name", svcName, "Sets the name of the service")
	flag.StringVar(&ctx.NatsUrl, "nats", "nats://localhost:4222", "URL to NATS service")
	flag.IntVar(&ctx.Port, "port", 3000, "Port for HTTP user interface, if supported by service")
	flag.StringVar(&ctx.Id, "id", "default", "Service instance identity. Important when running multiple instances of the same service")
	flag.Parse()

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

	reportInfo("starting engine %s", m.usvc.Name)
	go m.engine(m.usvc)

	reportInfo("entering service control loop")

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
				reportInfo(testOutput)
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
			default:
				reportError("unexpected control request #%d", c)
			}
		}
	}
	reportInfo("exiting service control loop")
	changes <- svc.Status{State: svc.StopPending}
	return
}

func RunService(usvc *DdUsvc, engine func(*DdUsvc)) {
	var err error
	reportInfo("starting %s service", usvc.Name)
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
		reportError("%s service failed: %s", usvc.Name, err.Error())
		return
	}
	reportInfo("%s service stopped", usvc.Name)
}
