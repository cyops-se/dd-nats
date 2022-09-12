package messages

import "dd-nats/inner/dd-nats-opcda/data"

// Common

type Tag struct {
	Tag string `json:"tag"`
}

type Groups struct {
	Items []data.OpcGroupItem `json:"items"`
}

// Requests
type GetOPCBranches struct {
	ServerId int    `json:"sid"`
	Branch   string `json:"branch"`
}

// Responses
type BrowserPosition struct {
	StatusResponse
	ServerId int      `json:"serverid"`
	Position string   `json:"position"`
	Branches []string `json:"branches"`
	Leaves   []string `json:"leaves"`
}

type OpcTagItemResponse struct {
	StatusResponse
	Items []data.OpcTagItem `json:"items"`
}

type OpcGroupItemResponse struct {
	StatusResponse
	Items []data.OpcGroupItem `json:"items"`
}
