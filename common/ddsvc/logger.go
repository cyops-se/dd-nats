package ddsvc

import (
	"dd-nats/common/types"
	"fmt"
	"log"
	"time"
)

func (svc *DdUsvc) Log(category string, title string, msg string) string {
	entry := &types.Log{Time: time.Now().UTC(), Category: category, Title: title, Description: msg}
	// if db.DB != nil {
	// 	db.DB.Create(&entry)
	// 	purge()
	// }
	text := fmt.Sprintf("%s: %s, %s", category, title, msg)
	log.Println(text)

	usvc.Publish("system.log."+category, entry)
	return text
}

func (svc *DdUsvc) Info(title string, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	text := svc.Log("info", title, msg)
	return fmt.Errorf(text)
}

func (svc *DdUsvc) Trace(title string, format string, args ...interface{}) error {
	if !svc.Context.Trace {
		return nil
	}

	msg := fmt.Sprintf(format, args...)
	text := svc.Log("trace", title, msg)
	return fmt.Errorf(text)
}

func (svc *DdUsvc) Error(title string, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	text := svc.Log("error", title, msg)
	// NotifySubscribers("ddsvc.Error", fmt.Sprintf("%s: %s", title, msg))
	return fmt.Errorf(text)
}

// func purge() {
// 	var result int64
// 	if db.DB != nil {
// 		db.DB.Model(&types.Log{}).Count(&result)
// 		for result > 1000 {
// 			var first types.Log
// 			db.DB.First(&first)
// 			db.DB.Unscoped().Delete(&first)
// 			result--
// 		}
// 	}
// }
