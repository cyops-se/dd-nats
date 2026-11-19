package main

import (
	"dd-nats/common/types"
	"testing"
	"time"
)

func countFiltered(out []publication) int {
	n := 0
	for _, p := range out {
		if p.topic == "process.filtered" {
			n++
		}
	}
	return n
}

func TestFilterNone(t *testing.T) {
	fp := &filteredPoint{FilterType: FilterTypeNone}
	out := filter(fp, types.DataPoint{Name: "a", Value: 1}, time.Now())
	if countFiltered(out) != 1 || len(out) != 3 {
		t.Fatalf("unexpected output: %v", out)
	}
}

func TestFilterInterval(t *testing.T) {
	fp := &filteredPoint{FilterType: FilterTypeInterval, Interval: 10}
	now := time.Now()

	if countFiltered(filter(fp, types.DataPoint{Value: 1}, now)) != 1 {
		t.Fatal("first value should pass")
	}
	if countFiltered(filter(fp, types.DataPoint{Value: 2}, now.Add(5*time.Second))) != 0 {
		t.Fatal("value within interval should be filtered")
	}
	if countFiltered(filter(fp, types.DataPoint{Value: 3}, now.Add(11*time.Second))) != 1 {
		t.Fatal("value after interval should pass")
	}
}

func TestFilterDeadband(t *testing.T) {
	// 10% of 0-100 => threshold 10
	fp := &filteredPoint{FilterType: FilterTypeDeadband, Deadband: 0.1, Min: 0, Max: 100}
	now := time.Now()

	values := []float64{5, 5, 5, 0, 0}
	// integrator: 5, 10, 15 (> 10 => pass, reset, prev = 5), -5, -10
	expected := []int{0, 0, 1, 0, 0}

	for i, v := range values {
		if got := countFiltered(filter(fp, types.DataPoint{Value: v}, now)); got != expected[i] {
			t.Fatalf("value %d (%v): got %d, expected %d (integrator %v)", i, v, got, expected[i], fp.Integrator)
		}
	}
}
