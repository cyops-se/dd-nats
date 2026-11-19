package modbus

import (
	"math"
	"testing"
)

func TestScaleValue(t *testing.T) {
	cases := []struct {
		item    ModbusItem
		raw     uint16
		value   float64
		quality int
	}{
		{ModbusItem{}, 1234, 1234, 192},                           // no scaling configured
		{ModbusItem{PlcRangeMin: 10, PlcRangeMax: 10}, 10, 10, 0}, // invalid range, no division by zero
		{ModbusItem{PlcRangeMin: 0, PlcRangeMax: 1000, RangeMin: 0, RangeMax: 100}, 500, 50, 192},
		{ModbusItem{PlcRangeMin: 100, PlcRangeMax: 200, RangeMin: 0, RangeMax: 10}, 50, -5, 0}, // below range, no uint16 underflow
		{ModbusItem{PlcRangeMin: 0, PlcRangeMax: 100, RangeMin: 0, RangeMax: 10}, 200, 20, 0},
	}

	for i, c := range cases {
		value, quality := scaleValue(&c.item, c.raw)
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value-c.value) > 1e-9 || quality != c.quality {
			t.Errorf("case %d: got %v/%d, expected %v/%d", i, value, quality, c.value, c.quality)
		}
	}
}

func TestBuildDatasets(t *testing.T) {
	slave := &ModbusSlaveItem{}
	var items []*ModbusItem

	// 130 consecutive holding registers => split in 125 + 5
	for i := 0; i < 130; i++ {
		items = append(items, &ModbusItem{FunctionCode: 3, ModbusAddress: uint(40001 + i)})
	}

	// a gap, then input registers
	items = append(items, &ModbusItem{FunctionCode: 3, ModbusAddress: 40200})
	items = append(items, &ModbusItem{FunctionCode: 4, ModbusAddress: 30001})
	items = append(items, &ModbusItem{FunctionCode: 4, ModbusAddress: 30002})

	ds := buildDatasets(slave, items)
	if len(ds) != 4 {
		t.Fatalf("expected 4 datasets, got %d", len(ds))
	}

	expected := []struct {
		fc, start, count uint16
	}{
		{3, 1, 125}, {3, 126, 5}, {3, 200, 1}, {4, 1, 2},
	}

	for i, e := range expected {
		if ds[i].fc != e.fc || ds[i].start != e.start || ds[i].count != e.count || int(ds[i].count) != len(ds[i].items) {
			t.Errorf("dataset %d: got fc %d start %d count %d items %d, expected %v", i, ds[i].fc, ds[i].start, ds[i].count, len(ds[i].items), e)
		}
	}
}
