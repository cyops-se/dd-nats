package routes

import "github.com/nats-io/nats.go"

func registerGroupRoutes(nc *nats.Conn) {
	nc.Subscribe("routes.groups.getall", getAllGroups)
}

func getAllGroups(msg *nats.Msg) {
	msg.Respond([]byte("get all groups"))
}
