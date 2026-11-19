package modbus

import (
	"dd-nats/common/db"
)

// GetModbusDataItems returns a snapshot of all configured items
func GetModbusDataItems() []*ModbusItem {
	lock.Lock()
	defer lock.Unlock()
	return append([]*ModbusItem(nil), datapoints...)
}

func AddModbusDataItem(item *ModbusItem) error {
	if err := db.DB.Create(item).Error; err != nil {
		return err
	}

	lock.Lock()
	datapoints = append(datapoints, item)
	lock.Unlock()
	return nil
}

func AddModbusDataItems(items ModbusItems) error {
	if len(items.Items) == 0 {
		return nil
	}

	if err := db.DB.Create(items.Items).Error; err != nil {
		return err
	}

	lock.Lock()
	datapoints = append(datapoints, items.Items...)
	lock.Unlock()
	return nil
}

// Must be called with lock held
func replaceItem(item *ModbusItem) {
	for i, dp := range datapoints {
		if item.ID == dp.ID {
			datapoints[i] = item
			return
		}
	}
}

func UpdateModbusItem(item *ModbusItem) error {
	if err := db.DB.Save(item).Error; err != nil {
		return err
	}

	lock.Lock()
	replaceItem(item)
	lock.Unlock()
	return nil
}

func UpdateModbusItems(items ModbusItems) error {
	if len(items.Items) == 0 {
		return nil
	}

	if err := db.DB.Save(&items.Items).Error; err != nil {
		return err
	}

	lock.Lock()
	for _, item := range items.Items {
		replaceItem(item)
	}
	lock.Unlock()
	return nil
}

func DeleteModbusItems(items ModbusItems) error {
	if len(items.Items) == 0 {
		return nil
	}

	if err := db.DB.Delete(items.Items).Error; err != nil {
		return err
	}

	deleted := make(map[uint]bool, len(items.Items))
	for _, i := range items.Items {
		deleted[i.ID] = true
	}

	lock.Lock()
	defer lock.Unlock()

	remaining := datapoints[:0]
	for _, dp := range datapoints {
		if !deleted[dp.ID] {
			remaining = append(remaining, dp)
		}
	}
	datapoints = remaining

	return nil
}

func BulkChangesModbusItems(items []*BulkChangeModbusItem) error {
	var anyerr error
	for _, posteditem := range items {
		slaveid, err := checkSlaveIP(posteditem)
		if err != nil {
			anyerr = err
			continue
		}

		var item ModbusItem
		err = db.DB.Table("modbus_items").Where("deleted_at is null").First(&item, "name = ?", posteditem.Name).Error

		item.Name = posteditem.Name
		item.Description = posteditem.Description
		item.EngUnit = posteditem.EngUnit
		item.DataType = posteditem.DataType
		item.DataLength = posteditem.DataLength
		item.ByteOrder = posteditem.ByteOrder
		item.ModbusAddress = posteditem.ModbusAddress
		item.FunctionCode = posteditem.FunctionCode
		item.RangeMin = posteditem.RangeMin
		item.RangeMax = posteditem.RangeMax
		item.PlcRangeMin = posteditem.PlcRangeMin
		item.PlcRangeMax = posteditem.PlcRangeMax
		item.ModbusSlaveID = slaveid

		if err == nil {
			if err = UpdateModbusItem(&item); err != nil {
				anyerr = err
			} else {
				usvc.Trace("Modbus TCP", "Bulk item saved: %s, %s", item.Name, item.Description)
			}
		} else {
			if err = AddModbusDataItem(&item); err != nil {
				anyerr = err
			} else {
				usvc.Trace("Modbus TCP", "Bulk item created: %s, %s", item.Name, item.Description)
			}
		}
	}

	return anyerr
}
