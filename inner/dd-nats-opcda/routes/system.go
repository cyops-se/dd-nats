package routes

func registerSystemRoutes() {
	usvc.Subscribe("system.heartbeat", systemHeartbeats)
}

func systemHeartbeats(topic string, responseTopic string, data []byte) error {
	// ddsvc.Trace("heartbeat received", "%s", string(msg.Data))
	return nil
}
