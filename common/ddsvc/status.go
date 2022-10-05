package ddsvc

type StatusResponse struct {
	Success       bool   `json:"success"`
	StatusMessage string `json:"statusmsg"`
}
