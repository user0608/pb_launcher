package systemstatus

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func collectPlatform(dataDir string, pids map[string]int) sample {
	s := sample{processes: make(map[string]processSample)}
	if data, err := os.ReadFile("/proc/stat"); err == nil {
		s.cpu = parseCPU(string(data))
	}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		s.memory = parseMemory(string(data))
	}
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if value, err := strconv.ParseFloat(fields[0], 64); err == nil && value >= 0 {
				s.uptime = &value
			}
		}
	}
	s.disk = collectDisk(dataDir)
	for id, pid := range pids {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		} // Exit or permissions may race with the PID snapshot.
		if proc, ok := parseProcess(string(data), pid); ok {
			s.processes[id] = proc
		}
	}
	return s
}

func parseCPU(data string) *cpuSample {
	s := &cpuSample{}
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] != "cpu" {
			if strings.HasPrefix(fields[0], "cpu") {
				s.cores++
			}
			continue
		}
		if len(fields) < 5 {
			return nil
		}
		// Guest and guest_nice (fields 9 and 10) are already counted in user/nice.
		for i := 1; i < len(fields) && i <= 8; i++ {
			value, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return nil
			}
			s.total += value
			if i == 4 || i == 5 {
				s.idle += value
			}
		}
	}
	if s.cores == 0 || s.total == 0 {
		return nil
	}
	return s
}

func parseMemory(data string) *MemoryInfo {
	values := make(map[string]uint64)
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[2] != "kB" {
			continue
		}
		if value, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			values[fields[0]] = value * 1024
		}
	}
	total, ok := values["MemTotal:"]
	available, hasAvailable := values["MemAvailable:"]
	if !ok || !hasAvailable || total == 0 || available > total {
		return nil
	}
	return &MemoryInfo{TotalBytes: total, UsedBytes: total - available}
}

func parseProcess(data string, pid int) (processSample, bool) {
	// comm may contain spaces and parentheses, so split after its closing ')'.
	end := strings.LastIndex(data, ")")
	if end < 0 {
		return processSample{}, false
	}
	fields := strings.Fields(data[end+1:])
	if len(fields) < 22 {
		return processSample{}, false
	}
	values := make([]uint64, 4)
	for i, index := range []int{11, 12, 19, 21} {
		value, err := strconv.ParseUint(fields[index], 10, 64)
		if err != nil {
			return processSample{}, false
		}
		values[i] = value
	}
	return processSample{pid: pid, ticks: values[0] + values[1], start: values[2], memory: values[3] * uint64(os.Getpagesize())}, true
}

func collectDisk(dataDir string) *DiskInfo {
	path, err := filepath.Abs(dataDir)
	if err != nil {
		return nil
	}
	// New installations may not have created data/ yet. Report its filesystem.
	for {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(path, &stat); err == nil {
			total := stat.Blocks * uint64(stat.Bsize)
			if total == 0 || stat.Bfree > stat.Blocks {
				return nil
			}
			return &DiskInfo{Path: path, TotalBytes: total, UsedBytes: (stat.Blocks - stat.Bfree) * uint64(stat.Bsize)}
		} else if !os.IsNotExist(err) {
			return nil
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}
