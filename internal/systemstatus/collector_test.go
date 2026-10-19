package systemstatus

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCollectorSnapshot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		current              sample
		wantCPU, wantProcess bool
	}{
		{"normal deltas", sample{cpu: &cpuSample{total: 2000, idle: 800, cores: 2}, processes: map[string]processSample{"a": {pid: 7, start: 1, ticks: 150, memory: 4096}}}, true, true},
		{"pid reused", sample{cpu: &cpuSample{total: 2000, idle: 800, cores: 2}, processes: map[string]processSample{"a": {pid: 7, start: 2, ticks: 150}}}, true, false},
		{"process exited", sample{cpu: &cpuSample{total: 2000, idle: 800, cores: 2}}, true, false},
		{"counters reset", sample{cpu: &cpuSample{total: 500, idle: 100, cores: 2}}, false, false},
		{"unsupported", sample{}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			collector := NewCollector("data", func() map[string]int { return map[string]int{"a": 7} })
			collector.collect = func(string, map[string]int) sample { calls++; return tc.current }
			collector.previous = sample{cpu: &cpuSample{total: 1000, idle: 400, cores: 2}, processes: map[string]processSample{"a": {pid: 7, start: 1, ticks: 100}}}
			status := collector.Snapshot()
			if (status.CPU.UsagePercent != nil) != tc.wantCPU {
				t.Fatalf("CPU availability: %+v", status.CPU)
			}
			if len(status.Instances) != 1 || status.ActiveInstances != 1 {
				t.Fatalf("instances: %+v", status)
			}
			proc := status.Instances[0]
			if (proc.CPUPercent != nil) != tc.wantProcess {
				t.Fatalf("process CPU availability: %+v", proc)
			}
			if tc.wantProcess && (*proc.CPUPercent != 10 || *proc.MemoryBytes != 4096) {
				t.Fatalf("process metrics: %+v", proc)
			}
			if tc.wantCPU && *status.CPU.UsagePercent != 60 {
				t.Fatalf("CPU: %v", *status.CPU.UsagePercent)
			}
			_ = collector.Snapshot()
			if calls != 1 {
				t.Fatalf("cached request collected %d times", calls)
			}
		})
	}
}

func TestCollectorFirstSampleAndRefresh(t *testing.T) {
	t.Parallel()
	collector := NewCollector("data", func() map[string]int { return map[string]int{} })
	collector.collect = func(string, map[string]int) sample { return sample{cpu: &cpuSample{total: 10, idle: 5, cores: 1}} }
	status := collector.Snapshot()
	if status.CPU.UsagePercent != nil || status.Instances == nil {
		t.Fatalf("first sample: %+v", status)
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"memory":null`) || !strings.Contains(string(encoded), `"instances":[]`) {
		t.Fatalf("unavailable contract: %s", encoded)
	}
	collector.status.SampledAt = time.Now().Add(-3 * time.Second)
	collector.collect = func(string, map[string]int) sample { return sample{cpu: &cpuSample{total: 20, idle: 10, cores: 1}} }
	status = collector.Snapshot()
	if status.CPU.UsagePercent == nil || *status.CPU.UsagePercent != 50 {
		t.Fatalf("second sample: %+v", status)
	}
}

func TestCollectorConcurrentReaders(t *testing.T) {
	t.Parallel()
	collector := NewCollector("data", func() map[string]int { return nil })
	calls := 0
	collector.collect = func(string, map[string]int) sample { calls++; return sample{} }
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); _ = collector.Snapshot() }()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("collected %d times", calls)
	}
}
