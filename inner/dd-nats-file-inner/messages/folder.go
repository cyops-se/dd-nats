package messages

import "dd-nats/common/ddsvc"

type FolderInfo struct {
	ddsvc.StatusResponse
	NewDir        string `json:"base"`
	ProcessingDir string `json:"processing"`
	DoneDir       string `json:"done"`
	FailDir       string `json:"fail"`
}
