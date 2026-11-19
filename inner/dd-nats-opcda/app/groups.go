package app

import (
	"dd-nats/common/db"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cyops-se/opc"
)

var usvc *ddsvc.DdUsvc
var opcmutex sync.Mutex // Issue #3, no time to find out where thread insafety is (looks like it's in or below oleutil)

// running keeps track of the groups with an active collector to prevent a
// group from being started more than once
var running = make(map[uint]bool)
var runningLock sync.Mutex

func read(client *opc.Connection) map[string]opc.Item {
	opcmutex.Lock()
	defer opcmutex.Unlock()
	return (*client).Read()
}

func addTag(client *opc.Connection, name string) error {
	opcmutex.Lock()
	defer opcmutex.Unlock()
	return (*client).AddSingle(name)
}

func asFloat64(v interface{}) float64 {
	invalidvalue := -9999.0
	switch i := v.(type) {
	case uint:
		return float64(i)
	case uint8:
		return float64(i)
	case uint16:
		return float64(i)
	case uint32:
		return float64(i)
	case uint64:
		return float64(i)
	case int:
		return float64(i)
	case int8:
		return float64(i)
	case int16:
		return float64(i)
	case int32:
		return float64(i)
	case int64:
		return float64(i)
	case float32:
		return float64(i)
	case float64:
		return i
	case bool:
		if i {
			return 1.0
		}
		return 0.0
	case string:
		return invalidvalue
	default:
		log.Printf("unknown type: %T", i)
	}

	return invalidvalue
}

func setRunning(id uint, value bool) bool {
	runningLock.Lock()
	defer runningLock.Unlock()

	if value && running[id] {
		return false
	}

	if value {
		running[id] = true
	} else {
		delete(running, id)
	}

	return true
}

func setGroupState(group *OpcGroupItem, state int) {
	group.State = state
	if err := db.DB.Model(group).Update("state", state).Error; err != nil {
		usvc.Error("OPC group", "Failed to update state of group %s: %s", group.Name, err.Error())
	}
}

func groupDataCollector(group *OpcGroupItem, tags []*OpcTagItem) {
	defer setRunning(group.ID, false)

	interval := group.Interval
	if interval < 1 {
		interval = 1
	}

	timer := time.NewTicker(time.Duration(interval) * time.Second)
	defer timer.Stop()

	opcmutex.Lock()
	client, err := opc.NewConnectionWithoutTags(group.ProgID, // ProgId
		[]string{"localhost"}, //  OPC servers nodes
	)
	opcmutex.Unlock()

	if err != nil {
		usvc.Error("Failed to connect OPC DA server", "Group: %s, progid: %s, err: %s", group.Name, group.ProgID, err.Error())
		setGroupState(group, GroupStateStopped)
		usvc.Event("group.failed", group)
		return
	}

	defer func() {
		opcmutex.Lock()
		client.Close()
		opcmutex.Unlock()
	}()

	// Adding items, the failed ones are retried every minute
	var failedtags []*OpcTagItem
	lastcheck := time.Now()
	for _, tag := range tags {
		if err := addTag(&client, tag.Name); err != nil {
			usvc.Log("warning", "Unable to collect tag", fmt.Sprintf("%s, group: %s, progid: %s", tag.Name, group.Name, group.ProgID))
			failedtags = append(failedtags, tag)
		}
	}

	if len(client.Tags()) == 0 {
		usvc.Error("No tags to collect", "Group %s WARNING: no tags added successfully, progid: %s", group.Name, group.ProgID)
	}

	// Initiate group running state
	if len(failedtags) > 0 {
		setGroupState(group, GroupStateRunningWithWarning)
		usvc.Info("OPC group warning", "Group %s WARNING", group.Name)
		usvc.Event(fmt.Sprintf("group.%d.warning", group.ID), group)
	} else {
		setGroupState(group, GroupStateRunning)
		usvc.Info("OPC group started", "Group %s STARTED", group.Name)
		usvc.Event(fmt.Sprintf("group.%d.started", group.ID), group)
	}

	usvc.Trace("Collecting tags", "%d tags from group: %s", len(client.Tags()), group.Name)

	group.Counter = 0
	for {
		if g, e := GetGroup(group.ID); e != nil || g.State == GroupStateStopped {
			usvc.Info("OPC group stopped", "Group %s STOPPED", group.Name)
			usvc.Event(fmt.Sprintf("group.%d.stopped", group.ID), g)
			break
		}

		items := read(&client)
		for k, v := range items {
			dp := types.DataPoint{Time: v.Timestamp, Name: k, Value: asFloat64(v.Value), Quality: int(v.Quality)}
			usvc.Publish("process.actual", dp)
		}

		group.LastRun = time.Now()
		group.Counter = group.Counter + uint64(len(items))
		usvc.Event(fmt.Sprintf("group.%d.updated", group.ID), group)

		<-timer.C

		// Try to add failed items every minute
		if len(failedtags) > 0 && time.Since(lastcheck) > time.Minute {
			stillfailing := failedtags[:0]
			for _, tag := range failedtags {
				if err := addTag(&client, tag.Name); err != nil {
					stillfailing = append(stillfailing, tag)
				}
			}
			failedtags = stillfailing
			lastcheck = time.Now()

			if len(failedtags) == 0 {
				setGroupState(group, GroupStateRunning)
				usvc.Info("OPC group items", "All failing items in group %s are now successfully read again", group.Name)
			}
		}
	}
}

func InitGroups(svc *ddsvc.DdUsvc) {
	usvc = svc
	usvc.Get("tagpathdelimiter", ".")

	items, err := GetGroups()
	if err != nil {
		usvc.Error("OPC groups", "Failed to load groups: %s", err.Error())
		return
	}

	for _, item := range items {
		setGroupState(item, GroupStateStopped)
		usvc.Event(fmt.Sprintf("group.%d.stopped", item.ID), item)

		if item.RunAtStart {
			StartGroup(item)
		}
	}
}

func GetGroups() ([]*OpcGroupItem, error) {
	var items []*OpcGroupItem
	if err := db.DB.Order("id").Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func GetGroup(id uint) (OpcGroupItem, error) {
	var item OpcGroupItem
	if err := db.DB.Take(&item, id).Error; err != nil {
		return item, err
	}

	return item, nil
}

func GetDefaultGroup() (*OpcGroupItem, error) {
	var item OpcGroupItem
	if err := db.DB.First(&item, "default_group = ?", true).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func GetGroupTags(id uint) ([]*OpcTagItem, error) {
	var items []*OpcTagItem
	if err := db.DB.Find(&items, "group_id = ?", id).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func StartGroup(group *OpcGroupItem) (err error) {
	tags, err := GetGroupTags(group.ID)
	if err != nil {
		return usvc.Error("OPC collection start failed", "Failed to get tags for group: %s (id: %d), error: %s", group.Name, group.ID, err.Error())
	}

	if len(tags) == 0 {
		return usvc.Error("OPC collection start failed", "No tags in group: %s (id: %d)", group.Name, group.ID)
	}

	// Make sure the group is not already running
	if !setRunning(group.ID, true) {
		return usvc.Error("OPC collection start failed", "Group already running, group: %s (id: %d)", group.Name, group.ID)
	}

	go groupDataCollector(group, tags)
	return nil
}

func StopGroup(group *OpcGroupItem) (err error) {
	runningLock.Lock()
	isRunning := running[group.ID]
	runningLock.Unlock()

	if !isRunning {
		return usvc.Error("OPC group", "Group not running, group: %s (id: %d)", group.Name, group.ID)
	}

	// The collector checks the state in the database and stops
	setGroupState(group, GroupStateStopped)
	return nil
}

func GetTagNames() ([]string, error) {
	var items []string
	if err := db.DB.Model(&OpcTagItem{}).Where("deleted_at is null").Pluck("name", &items).Error; err != nil {
		return nil, err
	}

	return items, nil
}
