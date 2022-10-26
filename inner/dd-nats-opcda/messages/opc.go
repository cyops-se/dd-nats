package messages

import (
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/app"
)

// Common

type Tag struct {
	Tag string `json:"tag"`
}

type Tags struct {
	Items []Tag `json:"items"`
}

type OpcItems struct {
	Items []app.OpcTagItem `json:"items"`
}

type Groups struct {
	Items []app.OpcGroupItem `json:"items"`
}

// Requests
type GetOPCBranches struct {
	ServerId int    `json:"sid"`
	Branch   string `json:"branch"`
}

// Responses
type BrowserPosition struct {
	types.StatusResponse
	ServerId int      `json:"serverid"`
	Position string   `json:"position"`
	Branches []string `json:"branches"`
	Leaves   []string `json:"leaves"`
}

type OpcTagItemResponse struct {
	types.StatusResponse
	Items []*app.OpcTagItem `json:"items"`
}

type OpcGroupItemResponse struct {
	types.StatusResponse
	Item app.OpcGroupItem `json:"item"`
}

type OpcGroupItemsResponse struct {
	types.StatusResponse
	Items []*app.OpcGroupItem `json:"items"`
}
