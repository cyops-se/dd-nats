package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/cyops-se/opc"
	"github.com/go-ole/go-ole"
	"github.com/nats-io/nats.go"
)

type Server struct {
	ID     int          `json:"id"`
	ProgID string       `json:"progid"`
	Cursor *ole.VARIANT `json:"-"`
}

var servers []*Server
var mutex sync.Mutex

// Service routes

func registerOpcRoutes() {
	logger.Info("OPC DA", "Registering OPC DA routes")

	// Server routes
	ddnats.Subscribe("usvc.opc.servers.getall", getAllOpcServers)
	ddnats.Subscribe("usvc.opc.servers.root", getOpcServerRoot)
	ddnats.Subscribe("usvc.opc.servers.getbranch", getOpcServerBranch)
}

func getAllOpcServers(msg *nats.Msg) {
	if servers == nil {
		initServers()
	}

	ddnats.Respond(msg, servers)
}

func getOpcServerRoot(nmsg *nats.Msg) {
	if servers == nil {
		initServers()
	}

	sid, err := getServerId(nmsg)
	if err != nil {
		logger.Error("OPC DA", "Failed to unmarshal server id: %s", err.Error())
		response := &types.PlainMessage{Message: "Failed to unmarshal server id"}
		ddnats.Respond(nmsg, response)
		return
	}

	browser, err := getBrowser(sid)
	if err != nil {
		logger.Error("OPC DA", "Failed to get server browser: %s", err.Error())
		response := &types.PlainMessage{Message: "Failed to get server browser"}
		ddnats.Respond(nmsg, response)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	response := &messages.BrowserPosition{}
	opc.MoveCursorHome(browser)
	response.Branches = opc.CursorListBranches(browser)
	response.Leaves = opc.CursorListLeaves(browser)
	response.Position = fmt.Sprintf("root.%s", opc.CursorPosition(browser))
	response.ServerId = sid

	ddnats.Respond(nmsg, response)
}

func getOpcServerBranch(nmsg *nats.Msg) {
	var msg messages.GetOPCBranches
	err := json.Unmarshal(nmsg.Data, &msg)
	if err != nil {
		logger.Error("OPC DA", "Failed to unmarshal message: %s", err.Error())
		response := &types.PlainMessage{Message: "Failed to unmarshal message"}
		ddnats.Respond(nmsg, response)
		return
	}

	browser, err := getBrowser(msg.ServerId)
	mutex.Lock()
	defer mutex.Unlock()

	var response messages.BrowserPosition

	opc.MoveCursorTo(browser, msg.Branch)
	response.Branches = opc.CursorListBranches(browser)
	response.Leaves = opc.CursorListLeaves(browser)
	response.Position = fmt.Sprintf("root.%s", opc.CursorPosition(browser))
	response.ServerId = msg.ServerId

	ddnats.Respond(nmsg, response)
}

// Helper methods

func initServers() {
	mutex.Lock()
	defer mutex.Unlock()

	logger.Info("OPC DA", "Enumerating OPC DA servers")
	// Test if we can connect Graybox.Simulator since we can't browse it
	i := 0
	if client, err := opc.NewConnection(
		"Graybox.Simulator.1",                              // ProgId
		[]string{"localhost"},                              //  OPC servers nodes
		[]string{"numeric.sin.int64", "numeric.saw.float"}, // slice of OPC tags
	); err == nil {
		logger.Info("OPC DA", "Adding Graybox.Simulator")
		servers = append(servers, &Server{ProgID: "Graybox.Simulator", ID: i})
		i++
		defer client.Close()
	}

	if ao := opc.NewAutomationObject(); ao != nil {
		serversfound := ao.GetOPCServers("localhost")
		logger.Log("trace", "OPC server init", fmt.Sprintf("Found %d server(s) on '%s':\n", len(serversfound), "localhost"))
		for _, server := range serversfound {
			logger.Log("trace", "OPC server found", server)
			logger.Info("OPC DA", "Adding %s", server)
			servers = append(servers, &Server{ProgID: server, ID: i})
			i++
		}
	} else {
		logger.Log("error", "OPC server init failure", "Unable to get new automation object")
	}
}

func getServerId(msg *nats.Msg) (int, error) {
	var intmsg types.IntMessage
	log.Println("msg.Data:", msg.Data)
	if err := json.Unmarshal(msg.Data, &intmsg); err != nil {
		return 0, err
	}

	return intmsg.Value, nil
}

func getServer(sid int) (*Server, error) {
	if sid < 0 || sid >= len(servers) {
		return nil, fmt.Errorf("no such server id: %d", sid)
	}

	return servers[sid], nil
}

func getBrowser(sid int) (*ole.VARIANT, error) {
	server, err := getServer(sid)
	if err != nil {
		logger.Error("Servers engine", "Failed to get server '%s', error: %s", sid, err)
		return nil, err
	}

	if server.Cursor == nil {
		mutex.Lock()
		server.Cursor, err = opc.CreateBrowserCursor(server.ProgID, []string{"localhost"})
		mutex.Unlock()
	}

	return server.Cursor, err
}
