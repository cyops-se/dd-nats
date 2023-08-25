package messages

import "dd-nats/common/types"

type FolderInfo struct {
	types.StatusResponse
	NewDir        string `json:"base"`
	ProcessingDir string `json:"processing"`
	DoneDir       string `json:"done"`
	FailDir       string `json:"fail"`
}

type Manifest struct {
	types.StatusResponse
	Manifest types.FileManifest `json:"manifest"`
}
