package types

type StatusResponse struct {
	Success       bool   `json:"success"`
	StatusMessage string `json:"statusmsg"`
}
