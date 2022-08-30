// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package ddsvc

import (
	"dd-nats/common/logger"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

type myservice struct {
	engine func()
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

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	defer handlePanic()

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	reportInfo("starting engine")
	go m.engine()

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
	changes <- svc.Status{State: svc.StopPending}
	return
}

func RunService(name string, engine func()) {
	var err error
	reportInfo("starting %s service", name)
	run := svc.Run

	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}
	if !inService {
		run = debug.Run
	}

	err = run(name, &myservice{engine: engine})
	if err != nil {
		reportError("%s service failed: %v", name, err)
		return
	}
	reportInfo("%s service stopped", name)
}
