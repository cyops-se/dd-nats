package modbus

import (
	"dd-nats/common/db"
)

func GetModbusDataItems() []*ModbusItem {
	return datapoints
}

func AddModbusDataItem(item *ModbusItem) error {
	err := db.DB.Create(item).Error
	if err == nil {
		datapoints = append(datapoints, item)
	}
	return err
}

func AddModbusDataItems(items ModbusItems) error {
	err := db.DB.Create(items.Items).Error
	if err == nil {
		datapoints = append(datapoints, items.Items...)
	}
	return err
}

func UpdateModbusItem(item *ModbusItem) error {
	err := db.DB.Save(&item).Error
	if err == nil {
		for i, dp := range datapoints {
			if item.ID == dp.ID {
				datapoints[i] = item
				break
			}
		}
	}

	return err
}

func UpdateModbusItems(items ModbusItems) error {
	err := db.DB.Save(&items.Items).Error
	if err == nil {
		for _, item := range items.Items {
			UpdateModbusItem(item)
		}
	}

	return err
}

func DeleteModbusItems(items ModbusItems) error {
	err := db.DB.Delete(items.Items).Error
	if err == nil {
		for si, dp := range datapoints {
			for _, i := range items.Items {
				if dp.ID == i.ID {
					datapoints = append(datapoints[:si], datapoints[si+1:]...)
					break
				}
			}
		}
	}
	return err
}

func BulkChangesModbusItems(items []*BulkChangeModbusItem) error {
	var anyerr error
	for _, posteditem := range items {
		slaveid, err := checkSlaveIP(posteditem)
		if err == nil {
			if posteditem.ModbusSlaveID == 0 {
				posteditem.ModbusSlaveID = slaveid
			}

			var item ModbusItem
			err := db.DB.Table("modbus_items").First(&item, "name = ?", posteditem.Name).Error

			item.Name = posteditem.Name
			item.Description = posteditem.Description
			item.DataType = posteditem.DataType
			item.DataLength = posteditem.DataLength
			item.ByteOrder = posteditem.ByteOrder
			item.ModbusAddress = posteditem.ModbusAddress
			item.FunctionCode = posteditem.FunctionCode
			item.RangeMin = posteditem.RangeMin
			item.RangeMax = posteditem.RangeMax
			item.PlcRangeMin = posteditem.PlcRangeMin
			item.PlcRangeMax = posteditem.PlcRangeMax
			item.ModbusSlaveID = posteditem.ModbusSlaveID

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
		} else {
			anyerr = err
		}
	}

	return anyerr
}
