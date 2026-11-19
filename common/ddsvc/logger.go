package ddsvc

import (
	"dd-nats/common/types"
	"errors"
	"fmt"
	"log"
	"time"
)

func (svc *DdUsvc) Log(category string, title string, msg string) string {
	entry := &types.Log{Time: time.Now().UTC(), Source: svc.Name, Category: category, Title: title, Description: msg}
	text := fmt.Sprintf("%s: %s, %s", category, title, msg)
	log.Println(text)

	svc.Publish("system.log."+category, entry)
	return text
}

// Info logs an informational message. The returned error contains the logged text.
func (svc *DdUsvc) Info(title string, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return errors.New(svc.Log("info", title, msg))
}

// Trace logs a trace message if tracing is enabled, otherwise it does nothing and returns nil.
func (svc *DdUsvc) Trace(title string, format string, args ...interface{}) error {
	if svc == nil || svc.Context == nil || !svc.Context.Trace {
		return nil
	}

	msg := fmt.Sprintf(format, args...)
	return errors.New(svc.Log("trace", title, msg))
}

// Error logs an error message. The returned error contains the logged text.
func (svc *DdUsvc) Error(title string, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return errors.New(svc.Log("error", title, msg))
}
