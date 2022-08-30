package types

type UdpStatistics struct {
	TotalMsg  uint64 `json:"totalmsg"`
	TotalPkts uint64 `json:"totalpkts"`
}

type Heartbeat struct {
	Hostname string `json:"hostname"`
	AppName  string `json:"appname"`
	Version  string `json:"version"`
}
