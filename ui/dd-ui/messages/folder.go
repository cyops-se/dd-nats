package messages

type FolderInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type FolderInfos struct {
	StatusResponse
	Items []FolderInfo `json:"items"`
}
