package logger

import (
	"dd-nats/common/db"
	"dd-nats/common/types"
	"fmt"
	"log"
	"time"
)

func Log(category string, title string, msg string) string {
	entry := &types.Log{Time: time.Now().UTC(), Category: category, Title: title, Description: msg}
	if db.DB != nil {
		db.DB.Create(&entry)
		purge()
	}
	text := fmt.Sprintf("%s: %s, %s", category, title, msg)
	log.Printf(text)
	return text
}

func Trace(title string, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	text := Log("trace", title, msg)
	return fmt.Errorf(text)
}

func Error(title string, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	text := Log("error", title, msg)
	// NotifySubscribers("logger.error", fmt.Sprintf("%s: %s", title, msg))
	return fmt.Errorf(text)
}

func purge() {
	var result int64
	if db.DB != nil {
		db.DB.Model(&types.Log{}).Count(&result)
		for result > 1000 {
			var first types.Log
			db.DB.First(&first)
			db.DB.Unscoped().Delete(&first)
			result--
		}
	}
}
