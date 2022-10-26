package messages

import "dd-nats/common/types"

type FolderInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type FolderInfos struct {
	types.StatusResponse
	Items []FolderInfo `json:"items"`
}
