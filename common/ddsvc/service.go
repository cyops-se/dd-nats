package ddsvc

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type DdUsvc struct {
	Name      string            `json:"name"`
	Context   *types.Context    `json:"context"`
	LastError error             `json:"lasterror"`
	Settings  map[string]string `json:"settings"`
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

func InitService(name string) *DdUsvc {
	if ctx := processArgs(name); ctx != nil {
		svc := &DdUsvc{Name: name, Context: ctx}
		svc.initSettings(ctx)

		ddnats.Connect(ctx.NatsUrl)
		shortname := strings.ReplaceAll(name, "-", "")
		ddnats.Subscribe("usvc."+shortname+"."+svc.Context.Id+".settings.get", svc.getSettings)
		ddnats.Subscribe("usvc."+shortname+"."+svc.Context.Id+".settings.set", svc.setSettings)
		ddnats.Subscribe("usvc."+shortname+"."+svc.Context.Id+".settings.delete", svc.deleteSetting)

		go svc.SendHeartbeat()

		return svc
	}

	return nil
}

func (svc *DdUsvc) RunService(engine func(*DdUsvc)) {
	RunService(svc, engine)
}

func (svc *DdUsvc) SendHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	hostname, _ := os.Hostname()

	for {
		<-ticker.C
		heartbeat := &types.Heartbeat{Hostname: hostname, AppName: svc.Name, Version: "0.0.1", Timestamp: time.Now().UTC(), Identity: svc.Context.Id}
		// payload, _ := json.Marshal(heartbeat)
		ddnats.Publish("system.heartbeat", heartbeat)
	}
}

func (svc *DdUsvc) RouteName(shortname string, method string) string {
	name := fmt.Sprintf("usvc.%s.%s.%s", shortname, svc.Context.Id, method)
	return name
}

func (svc *DdUsvc) Get(name string, defaultvalue string) string {
	if value, ok := svc.Settings[name]; ok {
		return value
	}

	svc.Set(name, defaultvalue)
	svc.saveSettings()

	return defaultvalue
}

func (svc *DdUsvc) GetInt(name string, defaultvalue int) int {
	if value, ok := svc.Settings[name]; ok {
		intvalue, _ := strconv.Atoi(value)
		return intvalue
	}

	svc.SetInt(name, defaultvalue)
	svc.saveSettings()

	return defaultvalue
}

func (svc *DdUsvc) Set(name string, value string) {
	svc.Settings[name] = value
	svc.saveSettings()
}

func (svc *DdUsvc) SetInt(name string, value int) {
	svc.Settings[name] = fmt.Sprintf("%d", value)
	svc.saveSettings()
}

// Internal service helpers
func (svc *DdUsvc) initSettings(ctx *types.Context) {
	filename := path.Join(svc.Context.Wdir, svc.Name+"_settings.json")
	if _, err := os.Stat(filename); err != nil {
		svc.Settings = make(map[string]string)
		svc.Settings["nats-url"] = svc.Context.NatsUrl
		svc.Settings["instance-id"] = svc.Context.Id
		if err = svc.saveSettings(); err != nil {
			log.Println("Failed to initialize settings:", err.Error())
		}
	}

	svc.loadSettings()
}

func (svc *DdUsvc) saveSettings() error {
	filename := path.Join(svc.Context.Wdir, svc.Name+"_settings.json")
	if content, err := json.Marshal(svc.Settings); err == nil {
		err := os.WriteFile(filename, content, 0755)
		ddnats.Publish(fmt.Sprintf("usvc.%s.event.settingschanged", strings.ReplaceAll(svc.Name, "-", "")), svc.Name)
		return err
	} else {
		return err
	}
}

func (svc *DdUsvc) loadSettings() error {
	filename := path.Join(svc.Context.Wdir, svc.Name+"_settings.json")
	if content, err := os.ReadFile(filename); err == nil {
		if err = json.Unmarshal(content, &svc.Settings); err == nil {
			// settings has precedence
			if url, ok := svc.Settings["nats-url"]; ok {
				svc.Context.NatsUrl = url
			}

			//command line has precedence
			if svc.Context.Id == "default" {
				if id, ok := svc.Settings["instance-id"]; ok {
					svc.Context.Id = id
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
func (svc *DdUsvc) getSettings(nmsg *nats.Msg) {
	// No arguments to request to unmarshal, continue to responding
	response := &GetSettingsResponse{Items: svc.Settings}
	response.Success = true
	ddnats.Respond(nmsg, response)
}

func (svc *DdUsvc) setSettings(nmsg *nats.Msg) {
	// Unmarshal set request
	var response types.StatusResponse
	request := &SetSettingsRequest{}
	if err := json.Unmarshal(nmsg.Data, request); err == nil {
		svc.Settings = request.Items
		if err = svc.saveSettings(); err == nil {
			response.Success = true
		} else {
			response.StatusMessage = err.Error()
		}
	} else {
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}

func (svc *DdUsvc) deleteSetting(nmsg *nats.Msg) {
	// Unmarshal set request
	var response types.StatusResponse
	request := &DeleteSettingRequest{}
	if err := json.Unmarshal(nmsg.Data, request); err == nil {
		if _, ok := svc.Settings[request.Item]; ok {
			delete(svc.Settings, request.Item)
			if err = svc.saveSettings(); err == nil {
				response.Success = true
			} else {
				response.StatusMessage = err.Error()
			}
		}
	} else {
		response.StatusMessage = err.Error()
	}

	ddnats.Respond(nmsg, response)
}
