package routes

import (
	"dd-nats/common/types"
	"dd-nats/inner/dd-nats-opcda/messages"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cyops-se/opc"
	"github.com/go-ole/go-ole"
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
	usvc.Info("OPC DA", "Registering OPC DA routes")

	// Server routes
	usvc.Subscribe(usvc.RouteName("opc", "servers.getall"), getAllOpcServers)
	usvc.Subscribe(usvc.RouteName("opc", "servers.root"), getOpcServerRoot)
	usvc.Subscribe(usvc.RouteName("opc", "servers.getbranch"), getOpcServerBranch)
}

func getAllOpcServers(topic string, responseTopic string, data []byte) error {
	if servers == nil {
		initServers()
	}

	return usvc.Publish(responseTopic, servers)
}

func getOpcServerRoot(topic string, responseTopic string, data []byte) error {
	if servers == nil {
		initServers()
	}

	sid, err := getServerId(data)
	if err != nil {
		usvc.Error("OPC DA", "Failed to unmarshal server id: %s", err.Error())
		response := &types.PlainMessage{Message: "Failed to unmarshal server id"}
		return usvc.Publish(responseTopic, response)
	}

	browser, err := getBrowser(sid)
	if err != nil {
		usvc.Error("OPC DA", "Failed to get server browser: %s", err.Error())
		response := &types.PlainMessage{Message: "Failed to get server browser"}
		return usvc.Publish(responseTopic, response)
	}

	mutex.Lock()
	defer mutex.Unlock()

	var response messages.BrowserPosition
	response.Success = true

	opc.MoveCursorHome(browser)
	response.Branches = opc.CursorListBranches(browser)
	response.Leaves = opc.CursorListLeaves(browser)
	response.Position = fmt.Sprintf("root.%s", opc.CursorPosition(browser))
	response.ServerId = sid

	return usvc.Publish(responseTopic, response)
}

func getOpcServerBranch(topic string, responseTopic string, data []byte) error {
	var msg messages.GetOPCBranches
	err := json.Unmarshal(data, &msg)
	if err != nil {
		usvc.Error("OPC DA", "Failed to unmarshal message: %s", err.Error())
		response := &types.PlainMessage{Message: "Failed to unmarshal message"}
		usvc.Publish(responseTopic, response)
		return err
	}

	browser, err := getBrowser(msg.ServerId)
	mutex.Lock()
	defer mutex.Unlock()

	var response messages.BrowserPosition
	response.Success = true

	opc.MoveCursorTo(browser, msg.Branch)
	response.Branches = opc.CursorListBranches(browser)
	response.Leaves = opc.CursorListLeaves(browser)
	response.Position = fmt.Sprintf("root.%s", opc.CursorPosition(browser))
	response.ServerId = msg.ServerId

	return usvc.Publish(responseTopic, response)
}

// Helper methods

func initServers() {
	mutex.Lock()
	defer mutex.Unlock()

	usvc.Info("OPC DA", "Enumerating OPC DA servers")
	// Test if we can connect Graybox.Simulator since we can't browse it
	i := 0
	if client, err := opc.NewConnection(
		"Graybox.Simulator.1",                              // ProgId
		[]string{"localhost"},                              //  OPC servers nodes
		[]string{"numeric.sin.int64", "numeric.saw.float"}, // slice of OPC tags
	); err == nil {
		usvc.Info("OPC DA", "Adding Graybox.Simulator")
		servers = append(servers, &Server{ProgID: "Graybox.Simulator", ID: i})
		i++
		defer client.Close()
	}

	if ao := opc.NewAutomationObject(); ao != nil {
		serversfound := ao.GetOPCServers("localhost")
		usvc.Log("trace", "OPC server init", fmt.Sprintf("Found %d server(s) on '%s':\n", len(serversfound), "localhost"))
		for _, server := range serversfound {
			usvc.Log("trace", "OPC server found", server)
			usvc.Info("OPC DA", "Adding %s", server)
			servers = append(servers, &Server{ProgID: server, ID: i})
			i++
		}
	} else {
		usvc.Log("error", "OPC server init failure", "Unable to get new automation object")
	}
}

func getServerId(data []byte) (int, error) {
	var intmsg types.IntMessage
	if err := json.Unmarshal(data, &intmsg); err != nil {
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
		usvc.Error("Servers engine", "Failed to get server '%s', error: %s", sid, err)
		return nil, err
	}

	if server.Cursor == nil {
		mutex.Lock()
		server.Cursor, err = opc.CreateBrowserCursor(server.ProgID, []string{"localhost"})
		mutex.Unlock()
	}

	return server.Cursor, err
}
