// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux
// +build linux

package ddsvc

import (
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"flag"
	"fmt"
	"log"
)

var SysInfo types.SystemInformation

func handlePanic() {
	if r := recover(); r != nil {
		// logger.Error("Windows service error", "Panic, recovery: %#v", r)
		log.Printf("Linux service panic, recovery: %#v", r)
		return
	}
}

func reportError(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	logger.Error("Linux service error", msg)
}

func reportInfo(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args...)
	logger.Trace("Linux service info", msg)
}

func processArgs(svcName string) *types.Context {
	ctx := &types.Context{}
	flag.StringVar(&ctx.Cmd, "cmd", "debug", "install/remove commands are not implemented in Linux!")
	flag.StringVar(&ctx.Wdir, "workdir", ".", "Sets the working directory for the process")
	flag.BoolVar(&ctx.Trace, "trace", false, "Prints traces from the application to the console")
	flag.BoolVar(&ctx.Version, "v", false, "Prints the commit hash and exits")
	flag.StringVar(&ctx.Name, "name", svcName, "Sets the name of the service")
	flag.StringVar(&ctx.NatsUrl, "nats", "nats://localhost:4222", "URL to NATS service")
	flag.IntVar(&ctx.Port, "port", 3000, "Port for HTTP user interface, if supported by service")
	flag.StringVar(&ctx.Id, "id", "default", "Service instance identity. Important when running multiple instances of the same service")
	flag.Parse()

	SysInfo.GitVersion = GitVersion
	SysInfo.GitCommit = GitCommit

	if ctx.Version {
		fmt.Printf("%s version %s, commit: %s, build time: %s\n", svcName, SysInfo.GitVersion, SysInfo.GitCommit, SysInfo.BuildTime)
		return nil
	}

	return ctx
}

func RunService(usvc *DdUsvc, engine func(*DdUsvc)) {
	reportInfo("starting %s service", usvc.Name)
	engine(usvc)
	select {}
	reportInfo("%s service stopped", usvc.Name)
}
