//go:build linux
// +build linux

package ddsvc

import (
	"dd-nats/common/types"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var SysInfo types.SystemInformation

func handlePanic() {
	if r := recover(); r != nil {
		log.Printf("Linux service panic, recovery: %#v", r)
	}
}

func processArgs(svcName string) *types.Context {
	ctx := &types.Context{}
	flag.StringVar(&ctx.Cmd, "cmd", "debug", "install/remove commands are not implemented in Linux!")
	flag.StringVar(&ctx.Wdir, "workdir", ".", "Sets the working directory for the process")
	flag.BoolVar(&ctx.Trace, "trace", false, "Prints traces from the application to the console")
	flag.BoolVar(&ctx.Version, "v", false, "Prints the commit hash and exits")
	flag.StringVar(&ctx.Name, "name", svcName, "Sets the name of the service")
	flag.StringVar(&ctx.Url, "url", "nats://localhost:4222", "URL to NATS service")
	flag.IntVar(&ctx.Port, "port", 3000, "Port for HTTP user interface, if supported by service")
	flag.StringVar(&ctx.Id, "id", "default", "Service instance identity. Important when running multiple instances of the same service")
	flag.StringVar(&ctx.BindIP, "bindip", "0.0.0.0", "IP of the interface to listen to")
	flag.Parse()

	SysInfo.GitVersion = GitVersion
	SysInfo.GitCommit = GitCommit
	SysInfo.BuildTime = BuildTime

	if ctx.Version {
		fmt.Printf("%s version %s, commit: %s, build time: %s\n", svcName, SysInfo.GitVersion, SysInfo.GitCommit, SysInfo.BuildTime)
		return nil
	}

	if ctx.Cmd == "install" || ctx.Cmd == "remove" {
		fmt.Printf("The '%s' command is not supported on Linux, use systemd instead\n", ctx.Cmd)
		return nil
	}

	return ctx
}

func RunService(usvc *DdUsvc, engine func(*DdUsvc)) {
	usvc.Info("Service engine", "Starting %s service", usvc.Name)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer handlePanic()
		engine(usvc)
	}()

	sig := <-signals
	usvc.Info("Service engine", "%s service stopping (signal: %s)", usvc.Name, sig.String())
	usvc.runStopHooks()
	usvc.Info("Service engine", "%s service stopped", usvc.Name)
}
