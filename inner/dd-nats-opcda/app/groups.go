package app

import (
	"dd-nats/common/db"
	"dd-nats/common/ddnats"
	"dd-nats/common/ddsvc"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cyops-se/opc"
)

var opcmutex sync.Mutex // Issue #3, no time to find out where thread insafety is (looks like it's in or below oleutil)
var failedtags []*OpcTagItem
var lastcheck time.Time

func read(client *opc.Connection) map[string]opc.Item {
	opcmutex.Lock()
	defer opcmutex.Unlock()
	return (*client).Read()
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

func groupDataCollector(group *OpcGroupItem, tags []*OpcTagItem) {
	timer := time.NewTicker(time.Duration(group.Interval) * time.Second)

	client, err := opc.NewConnectionWithoutTags(group.ProgID, // ProgId
		[]string{"localhost"}, //  OPC servers nodes
	)

	if err != nil {
		logger.Log("error", "Failed to connect OPC DA server", fmt.Sprintf("Group: %s, progid: %s, err: %s", group.Name, group.ProgID, err.Error()))
		ddnats.Event("group.failed", group)
		return
	}

	defer client.Close()

	// Adding items
	for _, tag := range tags {
		if err := client.AddSingle(tag.Name); err != nil {
			logger.Log("warning", "Unable to collect tag", fmt.Sprintf("%s, group: %s, progid: %s", tag.Name, group.Name, group.ProgID))
			failedtags = append(failedtags, tag)
			lastcheck = time.Now()
		}
	}

	if len(client.Tags()) == 0 {
		logger.Log("error", "No tags to collect", fmt.Sprintf("Group: %s, progid: %s", group.Name, group.ProgID))
		ddnats.Event("group.failed", group)
		return
	}

	// Initiate group running state
	if len(client.Tags()) != len(tags) {
		group.State = GroupStateRunningWithWarning
		logger.Info("OPC group warning", "Group %s WARNING", group.Name)
		ddnats.Event(fmt.Sprintf("group.%d.warning", group.ID), group)
	} else {
		group.State = GroupStateRunning
		logger.Info("OPC group started", "Group %s STARTED", group.Name)
		ddnats.Event(fmt.Sprintf("group.%d.started", group.ID), group)
	}

	logger.Log("trace", "Collecting tags", fmt.Sprintf("%d tags from group: %s", len(client.Tags()), group.Name))

	// items := read(&client) // This is only to get the number of items
	msg := &types.DataPointSample{Version: 3, Group: group.Name}
	msg.Points = make([]types.DataPoint, 10)

	group.Counter = 0
	db.DB.Save(group)

	var i, b int // golang always initialize to 0
	for {
		if g, e := GetGroup(group.ID); e == nil && g.State == GroupStateStopped {
			logger.Info("OPC group stopped", "Group %s STOPPED", group.Name)
			ddnats.Event(fmt.Sprintf("group.%d.stopped", group.ID), g)
			// ddnats.Event("group.stopped", group)
			break
		}

		items := read(&client)

		for k, v := range items {
			msg.Points[b].Time = v.Timestamp
			msg.Points[b].Name = k
			msg.Points[b].Value = asFloat64(v.Value)
			msg.Points[b].Quality = int(v.Quality)

			ddnats.Publish("process.actual", msg.Points[b])

			// Send batch when msg.Points is full (keep it small to avoid fragmentation)
			if b == len(msg.Points)-1 {
				// if err := ddnats.Publish("process.message", msg); err != nil {
				// 	log.Printf("Failed to publish sample message, error: %s", err.Error())
				// }

				b = 0
				msg.Sequence++
			} else {
				b++
			}
			i++
		}

		group.LastRun = time.Now()
		group.Counter = group.Counter + uint64(len(items))

		db.DB.Model(&group).Updates(OpcGroupItem{LastRun: group.LastRun, Counter: group.Counter})
		ddnats.Event(fmt.Sprintf("group.%d.updated", group.ID), group)

		<-timer.C

		// Try to add failed items every minute
		if len(failedtags) > 0 && group.LastRun.Sub(lastcheck) > time.Minute {
			for ti, tag := range failedtags {
				if err := client.AddSingle(tag.Name); err == nil {
					failedtags = append(failedtags[:ti], failedtags[ti+1:]...)
				}
			}

			lastcheck = time.Now()
			if len(failedtags) == 0 {
				group.State = GroupStateRunning
				logger.Info("OPC group items", "All failing items in group %s are now successfully read again", group.Name)
			}
		}
	}
}

func InitGroups(svc *ddsvc.DdUsvc) {
	svc.Get("tagpathdelimiter", ".")

	items, _ := GetGroups()
	for _, item := range items {
		item.State = GroupStateStopped
		ddnats.Event(fmt.Sprintf("group.%d.stopped", item.ID), item)
		db.DB.Save(item)

		if item.RunAtStart {
			StartGroup(item)
		}
	}
}

func GetGroups() ([]*OpcGroupItem, error) {
	var items []*OpcGroupItem
	db.DB.Order("id").Find(&items)
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
	if err := db.DB.First(&item, "default_group = 1").Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func GetGroupTags(id uint) ([]*OpcTagItem, error) {
	var items []*OpcTagItem
	if err := db.DB.Find(&items, "groupid = ?", id).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func StartGroup(group *OpcGroupItem) (err error) {
	// Make sure the group is not already running
	if group.State == GroupStateRunning || group.State == GroupStateRunningWithWarning {
		err = fmt.Errorf("Group already running, group: %s (id: %d)", group.Name, group.ID)
		logger.Log("error", "OPC collection start failed", err.Error())
		return
	}

	var tags []*OpcTagItem
	db.DB.Find(&tags, "group_id = ?", group.ID)
	if len(tags) <= 0 {
		err = fmt.Errorf("Group does not have any tags defined, group: %s (id: %d)", group.Name, group.ID)
		logger.Log("error", "OPC collection start failed", err.Error())
		return
	}

	go groupDataCollector(group, tags)

	return
}

func StopGroup(group *OpcGroupItem) (err error) {
	// Make sure the group is running
	if group.State == GroupStateStopped || group.State == GroupStateUnknown {
		return logger.Info("OPC group", "Group not running, group: %s (id: %d)", group.Name, group.ID)
	}

	// Stop collection go routine (unsafe)
	group.State = GroupStateStopped
	db.DB.Save(group)

	return
}

func GetTagNames() ([]string, error) {
	var items []string
	if err := db.DB.Table("opc_tags").Where("deleted_at is null").Pluck("Name", &items).Error; err != nil {
		return nil, err
	}

	return items, nil
}
