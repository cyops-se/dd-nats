package messages

type FolderInfo struct {
	StatusResponse
	NewDir        string `json:"base"`
	ProcessingDir string `json:"processing"`
	DoneDir       string `json:"done"`
	FailDir       string `json:"fail"`
}
