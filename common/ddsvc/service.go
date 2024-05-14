package ddsvc

import (
	"dd-nats/common/ddmb"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type DdUsvc struct {
	Name          string              `json:"name"`
	Context       *types.Context      `json:"context"`
	LastError     error               `json:"lasterror"`
	Settings      map[string]string   `json:"settings"`
	MessageBroker ddmb.IMessageBroker `json:"-"`
}

type SetSettingsRequest struct {
	Items map[string]string `json:"items"`
}

type GetSettingsResponse struct {
	types.StatusResponse
	Items map[string]string `json:"items"`
}

type DeleteSettingRequest struct {
	Item string `json:"item"`
}

type DeleteSettingResponse struct {
	types.StatusResponse
	Items map[string]string `json:"items"`
}

var GitVersion string
var GitCommit string
var BuildTime string
var usvc *DdUsvc

func InitService(name string) *DdUsvc {
	if ctx := processArgs(name); ctx != nil {
		usvc = &DdUsvc{Name: name, Context: ctx}
		usvc.initSettings(ctx)

		usvc.MessageBroker = ddmb.NewMessageBroker(ctx.Url)
		if usvc.MessageBroker == nil {
			log.Fatalf("Failed to create message broker from url: %s", ctx.Url)
		}

		usvc.MessageBroker.Connect(ctx.Url)
		shortname := strings.ReplaceAll(name, "-", "")
		usvc.Subscribe("usvc."+shortname+"."+usvc.Context.Id+".settings.get", usvc.getSettings)
		usvc.Subscribe("usvc."+shortname+"."+usvc.Context.Id+".settings.set", usvc.setSettings)
		usvc.Subscribe("usvc."+shortname+"."+usvc.Context.Id+".settings.delete", usvc.deleteSetting)

		go usvc.SendHeartbeat()

		return usvc
	}

	return nil
}

func (svc *DdUsvc) RunService(engine func(*DdUsvc)) {
	RunService(svc, engine)
}

func (svc *DdUsvc) SendHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	hostname, _ := os.Hostname()
	version := fmt.Sprintf("%s (%s)", SysInfo.GitVersion, SysInfo.GitCommit)

	for {
		<-ticker.C
		heartbeat := &types.Heartbeat{Hostname: hostname, AppName: usvc.Name, Version: version, Timestamp: time.Now().UTC(), Identity: usvc.Context.Id}
		// payload, _ := json.Marshal(heartbeat)
		usvc.Publish("system.heartbeat", heartbeat)
	}
}

func (svc *DdUsvc) Publish(topic string, data interface{}) error {
	if usvc.MessageBroker != nil {
		return usvc.MessageBroker.Publish(topic, data)
	}

	return fmt.Errorf("unable to publish message, broker not initialized")
}

func (svc *DdUsvc) Request(topic string, data interface{}) ([]byte, error) {
	if usvc.MessageBroker != nil {
		return usvc.MessageBroker.Request(topic, data)
	}

	return nil, fmt.Errorf("unable to publish message, broker not initialized")
}

func (svc *DdUsvc) Subscribe(topic string, callback ddmb.IMessageHandler) error {
	if usvc.MessageBroker != nil {
		svc.Trace("Message broker", "Registering subscription on topic: %s", topic)
		return usvc.MessageBroker.Subscribe(topic, callback)
	}

	return fmt.Errorf("unable to subscribe topic, broker not initialized")
}

func (svc *DdUsvc) Event(subject string, arg interface{}) error {
	return svc.Publish("system.event."+subject, arg)
}

func (svc *DdUsvc) RouteName(shortname string, method string) string {
	name := fmt.Sprintf("usvc.%s.%s.%s", shortname, usvc.Context.Id, method)
	return name
}

func (svc *DdUsvc) Get(name string, defaultvalue string) string {
	if value, ok := usvc.Settings[name]; ok {
		return value
	}

	usvc.Set(name, defaultvalue)
	usvc.saveSettings()

	return defaultvalue
}

func (svc *DdUsvc) GetInt(name string, defaultvalue int) int {
	if value, ok := usvc.Settings[name]; ok {
		intvalue, _ := strconv.Atoi(value)
		return intvalue
	}

	usvc.SetInt(name, defaultvalue)
	usvc.saveSettings()

	return defaultvalue
}

func (svc *DdUsvc) Set(name string, value string) {
	usvc.Settings[name] = value
	usvc.saveSettings()
}

func (svc *DdUsvc) SetInt(name string, value int) {
	usvc.Settings[name] = fmt.Sprintf("%d", value)
	usvc.saveSettings()
}

// Internal service helpers
func (svc *DdUsvc) initSettings(ctx *types.Context) {
	filename := path.Join(usvc.Context.Wdir, usvc.Name+"_settings.json")
	if _, err := os.Stat(filename); err != nil {
		usvc.Settings = make(map[string]string)
		usvc.Settings["url"] = usvc.Context.Url
		usvc.Settings["instance-id"] = usvc.Context.Id
		if err = usvc.saveSettings(); err != nil {
			log.Println("Failed to initialize settings:", err.Error())
		}
	}

	usvc.loadSettings()
}

func (svc *DdUsvc) saveSettings() error {
	filename := path.Join(usvc.Context.Wdir, usvc.Name+"_settings.json")
	if content, err := json.Marshal(usvc.Settings); err == nil {
		err := os.WriteFile(filename, content, 0755)
		usvc.Publish(fmt.Sprintf("usvc.%s.event.settingschanged", strings.ReplaceAll(usvc.Name, "-", "")), usvc.Name)
		return err
	} else {
		return err
	}
}

func (svc *DdUsvc) loadSettings() error {
	filename := path.Join(usvc.Context.Wdir, usvc.Name+"_settings.json")
	if content, err := os.ReadFile(filename); err == nil {
		if err = json.Unmarshal(content, &usvc.Settings); err == nil {
			// command line argument have precedence
			if usvc.Context.Url == "nats://localhost:4222" {
				if url, ok := usvc.Settings["url"]; ok {
					usvc.Context.Url = url
				}
			}

			//command line argument has precedence
			if usvc.Context.Id == "default" {
				if id, ok := usvc.Settings["instance-id"]; ok {
					usvc.Context.Id = id
				}
			}

			return nil
		}

		return err
	} else {
		return err
	}
}

// Service methods (NATS providers)
func (svc *DdUsvc) getSettings(topic string, responseTopic string, data []byte) error {
	// No arguments to request to unmarshal, continue to responding
	response := &GetSettingsResponse{Items: usvc.Settings}
	response.Success = true
	usvc.Publish(responseTopic, response)
	return nil
}

func (svc *DdUsvc) setSettings(topic string, responseTopic string, data []byte) error {
	// Unmarshal set request
	var response types.StatusResponse
	request := &SetSettingsRequest{}
	if err := json.Unmarshal(data, request); err == nil {
		usvc.Settings = request.Items
		if err = usvc.saveSettings(); err == nil {
			response.Success = true
		} else {
			response.StatusMessage = err.Error()
		}
	} else {
		response.StatusMessage = err.Error()
	}

	usvc.Publish(responseTopic, response)
	return nil
}

func (svc *DdUsvc) deleteSetting(topic string, responseTopic string, data []byte) error {
	// Unmarshal set request
	var response types.StatusResponse
	request := &DeleteSettingRequest{}
	if err := json.Unmarshal(data, request); err == nil {
		if _, ok := usvc.Settings[request.Item]; ok {
			delete(usvc.Settings, request.Item)
			if err = usvc.saveSettings(); err == nil {
				response.Success = true
			} else {
				response.StatusMessage = err.Error()
			}
		}
	} else {
		response.StatusMessage = err.Error()
	}

	usvc.Publish(responseTopic, response)
	return nil
}
