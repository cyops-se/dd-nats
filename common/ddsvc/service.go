package ddsvc

import (
	"dd-nats/common/ddmb"
	"dd-nats/common/types"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DdUsvc struct {
	Name          string              `json:"name"`
	Context       *types.Context      `json:"context"`
	LastError     error               `json:"lasterror"`
	Settings      map[string]string   `json:"settings"`
	MessageBroker ddmb.IMessageBroker `json:"-"`

	settingsLock sync.RWMutex
	stopLock     sync.Mutex
	stopHooks    []func()
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
	ctx := processArgs(name)
	if ctx == nil {
		return nil
	}

	finalizeContext(ctx)

	svc := &DdUsvc{Name: name, Context: ctx}
	usvc = svc
	svc.initSettings()

	svc.MessageBroker = ddmb.NewMessageBroker(ctx.Url)
	if svc.MessageBroker == nil {
		log.Fatalf("Failed to create message broker from url: %s", ctx.Url)
	}

	if err := svc.MessageBroker.Connect(ctx.Url); err != nil {
		log.Fatalf("Failed to connect message broker, url: %s, error: %s", ctx.Url, err.Error())
	}

	shortname := strings.ReplaceAll(name, "-", "")
	svc.Subscribe("usvc."+shortname+"."+svc.Context.Id+".settings.get", svc.getSettings)
	svc.Subscribe("usvc."+shortname+"."+svc.Context.Id+".settings.set", svc.setSettings)
	svc.Subscribe("usvc."+shortname+"."+svc.Context.Id+".settings.delete", svc.deleteSetting)

	go svc.SendHeartbeat()

	return svc
}

// finalizeContext records which flags were explicitly set and makes the working
// directory absolute and current, so that all relative paths used by the
// services end up in the working directory (also when running as a Windows
// service, where the default current directory is C:\Windows\System32)
func finalizeContext(ctx *types.Context) {
	ctx.SetFlags = make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { ctx.SetFlags[f.Name] = true })

	if ctx.Wdir == "" {
		ctx.Wdir = "."
	}

	if abs, err := filepath.Abs(ctx.Wdir); err == nil {
		ctx.Wdir = abs
	}

	if err := os.MkdirAll(ctx.Wdir, 0755); err != nil {
		log.Printf("Failed to create working directory %s: %s", ctx.Wdir, err.Error())
	}

	if err := os.Chdir(ctx.Wdir); err != nil {
		log.Printf("Failed to change to working directory %s: %s", ctx.Wdir, err.Error())
	}
}

func (svc *DdUsvc) RunService(engine func(*DdUsvc)) {
	RunService(svc, engine)
}

// OnStop registers a function that is called when the service is stopped.
// Hooks are called in reverse order of registration.
func (svc *DdUsvc) OnStop(hook func()) {
	svc.stopLock.Lock()
	defer svc.stopLock.Unlock()
	svc.stopHooks = append(svc.stopHooks, hook)
}

func (svc *DdUsvc) runStopHooks() {
	svc.stopLock.Lock()
	hooks := svc.stopHooks
	svc.stopHooks = nil
	svc.stopLock.Unlock()

	for i := len(hooks) - 1; i >= 0; i-- {
		func() {
			defer handlePanic()
			hooks[i]()
		}()
	}

	if svc.MessageBroker != nil {
		svc.MessageBroker.Disconnect()
	}
}

func (svc *DdUsvc) SendHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	hostname, _ := os.Hostname()
	version := fmt.Sprintf("%s (%s)", SysInfo.GitVersion, SysInfo.GitCommit)

	for range ticker.C {
		heartbeat := &types.Heartbeat{Hostname: hostname, AppName: svc.Name, Version: version, Timestamp: time.Now().UTC(), Identity: svc.Context.Id}
		svc.Publish("system.heartbeat", heartbeat)
	}
}

func (svc *DdUsvc) Publish(topic string, data interface{}) error {
	if svc.MessageBroker != nil {
		return svc.MessageBroker.Publish(topic, data)
	}

	return fmt.Errorf("unable to publish message, broker not initialized")
}

func (svc *DdUsvc) Request(topic string, data interface{}) ([]byte, error) {
	if svc.MessageBroker != nil {
		return svc.MessageBroker.Request(topic, data)
	}

	return nil, fmt.Errorf("unable to publish message, broker not initialized")
}

func (svc *DdUsvc) Subscribe(topic string, callback ddmb.IMessageHandler) error {
	if svc.MessageBroker == nil {
		return fmt.Errorf("unable to subscribe topic, broker not initialized")
	}

	svc.Trace("Message broker", "Registering subscription on topic: %s", topic)
	if err := svc.MessageBroker.Subscribe(topic, callback); err != nil {
		return svc.Error("Message broker", "Failed to subscribe topic: %s, error: %s", topic, err.Error())
	}

	return nil
}

func (svc *DdUsvc) Unsubscribe(topic string) error {
	if svc.MessageBroker == nil {
		return fmt.Errorf("unable to unsubscribe topic, broker not initialized")
	}

	svc.Trace("Message broker", "Removing subscription on topic: %s", topic)
	return svc.MessageBroker.Unsubscribe(topic)
}

func (svc *DdUsvc) Event(subject string, arg interface{}) error {
	return svc.Publish("system.event."+subject, arg)
}

func (svc *DdUsvc) RouteName(shortname string, method string) string {
	return fmt.Sprintf("usvc.%s.%s.%s", shortname, svc.Context.Id, method)
}

// Get returns the named setting, or stores and returns the default value if it doesn't exist
func (svc *DdUsvc) Get(name string, defaultvalue string) string {
	svc.settingsLock.RLock()
	value, ok := svc.Settings[name]
	svc.settingsLock.RUnlock()

	if ok {
		return value
	}

	svc.Set(name, defaultvalue)
	return defaultvalue
}

// GetInt returns the named setting as an integer, or stores and returns the default value
// if it doesn't exist. If the stored value isn't a valid integer, the default value is returned.
func (svc *DdUsvc) GetInt(name string, defaultvalue int) int {
	svc.settingsLock.RLock()
	value, ok := svc.Settings[name]
	svc.settingsLock.RUnlock()

	if !ok {
		svc.SetInt(name, defaultvalue)
		return defaultvalue
	}

	intvalue, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		log.Printf("Setting '%s' has invalid integer value '%s', using default %d", name, value, defaultvalue)
		return defaultvalue
	}

	return intvalue
}

func (svc *DdUsvc) Set(name string, value string) {
	svc.settingsLock.Lock()
	svc.Settings[name] = value
	svc.settingsLock.Unlock()

	if err := svc.saveSettings(); err != nil {
		log.Printf("Failed to save settings: %s", err.Error())
	}
}

func (svc *DdUsvc) SetInt(name string, value int) {
	svc.Set(name, strconv.Itoa(value))
}

// Internal service helpers
func (svc *DdUsvc) settingsFilename() string {
	return filepath.Join(svc.Context.Wdir, svc.Name+"_settings.json")
}

func (svc *DdUsvc) initSettings() {
	svc.Settings = make(map[string]string)

	filename := svc.settingsFilename()
	if _, err := os.Stat(filename); err != nil {
		svc.Settings["url"] = svc.Context.Url
		svc.Settings["instance-id"] = svc.Context.Id
		if err = svc.writeSettings(); err != nil {
			log.Println("Failed to initialize settings:", err.Error())
		}
		return
	}

	if err := svc.loadSettings(); err != nil {
		log.Printf("Failed to load settings from %s: %s", filename, err.Error())
	}
}

// writeSettings writes the settings to file without publishing a change event
func (svc *DdUsvc) writeSettings() error {
	svc.settingsLock.RLock()
	content, err := json.MarshalIndent(svc.Settings, "", "  ")
	svc.settingsLock.RUnlock()

	if err != nil {
		return err
	}

	// Write to a temporary file first to avoid a corrupt settings file if the process dies while writing
	filename := svc.settingsFilename()
	tmpname := filename + ".tmp"
	if err = os.WriteFile(tmpname, content, 0644); err != nil {
		return err
	}

	return os.Rename(tmpname, filename)
}

func (svc *DdUsvc) saveSettings() error {
	if err := svc.writeSettings(); err != nil {
		return err
	}

	svc.Publish(fmt.Sprintf("usvc.%s.event.settingschanged", strings.ReplaceAll(svc.Name, "-", "")), svc.Name)
	return nil
}

func (svc *DdUsvc) loadSettings() error {
	content, err := os.ReadFile(svc.settingsFilename())
	if err != nil {
		return err
	}

	settings := make(map[string]string)
	if err = json.Unmarshal(content, &settings); err != nil {
		return err
	}

	svc.settingsLock.Lock()
	svc.Settings = settings
	svc.settingsLock.Unlock()

	// command line arguments have precedence
	if !svc.Context.FlagSet("url") {
		if url, ok := settings["url"]; ok && url != "" {
			svc.Context.Url = url
		}
	}

	if !svc.Context.FlagSet("id") {
		if id, ok := settings["instance-id"]; ok && id != "" {
			svc.Context.Id = id
		}
	}

	return nil
}

func (svc *DdUsvc) copySettings() map[string]string {
	svc.settingsLock.RLock()
	defer svc.settingsLock.RUnlock()

	items := make(map[string]string, len(svc.Settings))
	for k, v := range svc.Settings {
		items[k] = v
	}

	return items
}

// Service methods (NATS providers)
func (svc *DdUsvc) getSettings(topic string, responseTopic string, data []byte) error {
	// No arguments to request to unmarshal, continue to responding
	response := &GetSettingsResponse{Items: svc.copySettings()}
	response.Success = true
	return svc.Publish(responseTopic, response)
}

func (svc *DdUsvc) setSettings(topic string, responseTopic string, data []byte) error {
	var response types.StatusResponse
	request := &SetSettingsRequest{}
	if err := json.Unmarshal(data, request); err != nil {
		response.StatusMessage = err.Error()
	} else if request.Items == nil {
		response.StatusMessage = "no settings items provided"
	} else {
		svc.settingsLock.Lock()
		svc.Settings = request.Items
		svc.settingsLock.Unlock()

		if err = svc.saveSettings(); err == nil {
			response.Success = true
		} else {
			response.StatusMessage = err.Error()
		}
	}

	return svc.Publish(responseTopic, response)
}

func (svc *DdUsvc) deleteSetting(topic string, responseTopic string, data []byte) error {
	var response types.StatusResponse
	request := &DeleteSettingRequest{}
	if err := json.Unmarshal(data, request); err != nil {
		response.StatusMessage = err.Error()
	} else {
		svc.settingsLock.Lock()
		_, ok := svc.Settings[request.Item]
		delete(svc.Settings, request.Item)
		svc.settingsLock.Unlock()

		if !ok {
			response.StatusMessage = fmt.Sprintf("no such setting: %s", request.Item)
		} else if err = svc.saveSettings(); err == nil {
			response.Success = true
		} else {
			response.StatusMessage = err.Error()
		}
	}

	return svc.Publish(responseTopic, response)
}
