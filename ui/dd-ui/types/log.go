package types

import (
	"time"

	"gorm.io/gorm"
)

type Log struct {
	gorm.Model
	Time        time.Time `json:"time"`
	Source      string    `json:"source"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}
