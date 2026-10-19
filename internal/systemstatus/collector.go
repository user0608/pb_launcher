package systemstatus

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

// Nil measurements are unavailable, never an invented zero. CPU needs two
// samples; process CPU uses 100% for one fully utilized logical core.
type CPUInfo struct {
	UsagePercent *float64 `json:"usage_percent"`
	Cores        int      `json:"cores"`
}

type MemoryInfo struct {
	TotalBytes uint64 `json:"total_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
}

type DiskInfo struct {
	TotalBytes uint64 `json:"total_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
	Path       string `json:"path"`
}

type InstanceInfo struct {
	ServiceID   string   `json:"service_id"`
	CPUPercent  *float64 `json:"cpu_percent"`
	MemoryBytes *uint64  `json:"memory_bytes"`
}

type Status struct {
	SampledAt       time.Time      `json:"sampled_at"`
	OS              string         `json:"os"`
	CPU             CPUInfo        `json:"cpu"`
	Memory          *MemoryInfo    `json:"memory"`
	Disk            *DiskInfo      `json:"disk"`
	UptimeSeconds   *float64       `json:"uptime_seconds"`
	ActiveInstances int            `json:"active_instances"`
	Instances       []InstanceInfo `json:"instances"`
}

type cpuSample struct {
	total, idle uint64
	cores       int
}
type processSample struct {
	pid          int
	start, ticks uint64
	memory       uint64
}
type sample struct {
	cpu       *cpuSample
	memory    *MemoryInfo
	disk      *DiskInfo
	uptime    *float64
	processes map[string]processSample
}

type Collector struct {
	mu       sync.Mutex
	pids     func() map[string]int
	collect  func(string, map[string]int) sample
	dataDir  string
	previous sample
	status   Status
}

func NewCollector(dataDir string, pids func() map[string]int) *Collector {
	return &Collector{dataDir: dataDir, pids: pids, collect: collectPlatform}
}

// Snapshot samples on demand, at most once every two seconds across clients.
// It creates no background goroutines and retains no exited-process history.
func (c *Collector) Snapshot() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.status.SampledAt.IsZero() && time.Since(c.status.SampledAt) < 2*time.Second {
		return c.status
	}
	pids := c.pids()
	next := c.collect(c.dataDir, pids)
	status := Status{
		SampledAt: time.Now().UTC(), OS: runtime.GOOS,
		CPU: CPUInfo{Cores: runtime.NumCPU()}, Memory: next.memory,
		Disk: next.disk, UptimeSeconds: next.uptime,
		ActiveInstances: len(pids), Instances: make([]InstanceInfo, 0, len(pids)),
	}
	var delta uint64
	if next.cpu != nil {
		status.CPU.Cores = next.cpu.cores
		prev := c.previous.cpu
		if prev != nil && prev.cores == next.cpu.cores && next.cpu.total > prev.total && next.cpu.idle >= prev.idle {
			delta = next.cpu.total - prev.total
			idle := next.cpu.idle - prev.idle
			if idle <= delta {
				usage := 100 * float64(delta-idle) / float64(delta)
				status.CPU.UsagePercent = &usage
			}
		}
	}
	for id := range pids {
		info := InstanceInfo{ServiceID: id}
		if proc, ok := next.processes[id]; ok {
			memory := proc.memory
			info.MemoryBytes = &memory
			prev, exists := c.previous.processes[id]
			if exists && delta > 0 && proc.pid == prev.pid && proc.start == prev.start && proc.ticks >= prev.ticks {
				usage := 100 * float64(proc.ticks-prev.ticks) * float64(status.CPU.Cores) / float64(delta)
				info.CPUPercent = &usage
			}
		}
		status.Instances = append(status.Instances, info)
	}
	sort.Slice(status.Instances, func(i, j int) bool { return status.Instances[i].ServiceID < status.Instances[j].ServiceID })
	c.previous, c.status = next, status
	return status
}
