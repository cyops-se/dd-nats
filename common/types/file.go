package types

import (
	"time"

	"gorm.io/gorm"
)

type FolderInfo struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
}

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

type FileTransferStart struct {
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Size          int       `json:"size"`
	FileTime      time.Time `json:"filetime"`
	TransferStart time.Time `json:"transferstart"`
	TransferId    string    `json:"id"`
}

type FileTransferBlock struct {
	TransferId string `json:"id"`
	BlockNo    uint64 `json:"blockno"`
	FileIndex  uint64 `json:"fileindex"`
	Size       uint64 `json:"size"`
	Payload    []byte `json:"payload"`
}

type FileTransferEnd struct {
	TransferId  string `json:"id"`
	TotalBlocks uint64 `json:"totalblocks"`
	TotalSize   uint64 `json:"totalsize"`
	Error       string `json:"error"`
	Hash        []byte `json:"hash"`
}

type FileProgress struct {
	File        *FileInfo `json:"file"`
	TotalSent   int       `json:"totalsent"`
	PercentDone float64   `json:"percentdone"`
}

type FileTransferInfo struct {
	SentFiles []FileInfo `json:"sentfiles"`
}
