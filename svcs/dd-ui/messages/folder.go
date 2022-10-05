package messages

import "dd-nats/common/ddsvc"

type FolderInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type FolderInfos struct {
	ddsvc.StatusResponse
	Items []FolderInfo `json:"items"`
}
