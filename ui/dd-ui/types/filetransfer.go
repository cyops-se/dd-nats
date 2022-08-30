package types

import (
	"time"

	"gorm.io/gorm"
)

type FileTransferConfig struct {
	gorm.Model
	NewDirectory      string `json:"newdir"`
	ProgressDirectory string `json:"progressdir"`
	DoneDirectory     string `json:"donedir"`
	ChunkSize         int    `json:"chunksize"`
	ChunksDelay       int    `json:"chunkdelay"`    // in milliseconds
	RetentionTime     int    `json:"retentiontime"` // in days
}

type FileInfo struct {
	Name string    `json:"name"`
	Path string    `json:"path"`
	Size int       `json:"size"`
	Date time.Time `json:"time"`
}

type FileProgress struct {
	File        *FileInfo `json:"file"`
	TotalSent   int       `json:"totalsent"`
	PercentDone float64   `json:"percentdone"`
}

type FileTransferInfo struct {
	SentFiles []FileInfo `json:"sentfiles"`
}
