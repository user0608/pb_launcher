package systemstatus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCPU(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		valid       bool
		total, idle uint64
	}{
		{"guest not double counted", "cpu 100 20 30 400 50 6 7 8 9 10\ncpu0 1\ncpu1 1\n", true, 621, 450},
		{"truncated", "cpu 1 2\ncpu0 1", false, 0, 0},
		{"invalid", "cpu bad 0 0 0\ncpu0 1", false, 0, 0},
		{"empty", "", false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := parseCPU(tc.input)
			if (value != nil) != tc.valid {
				t.Fatalf("sample=%+v", value)
			}
			if value != nil && (value.total != tc.total || value.idle != tc.idle || value.cores != 2) {
				t.Fatalf("sample=%+v", value)
			}
		})
	}
}

func TestParseMemory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		valid       bool
	}{
		{"available includes reclaimable cache", "MemTotal: 100 kB\nMemAvailable: 60 kB\nMemFree: 10 kB", true},
		{"missing available", "MemTotal: 100 kB", false},
		{"impossible", "MemTotal: 100 kB\nMemAvailable: 101 kB", false},
		{"zero total", "MemTotal: 0 kB\nMemAvailable: 0 kB", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := parseMemory(tc.input)
			if (value != nil) != tc.valid {
				t.Fatalf("memory=%+v", value)
			}
			if value != nil && (value.TotalBytes != 102400 || value.UsedBytes != 40960) {
				t.Fatalf("memory=%+v", value)
			}
		})
	}
}

func TestParseProcess(t *testing.T) {
	t.Parallel()
	fields := make([]string, 22)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0], fields[11], fields[12], fields[19], fields[21] = "S", "10", "20", "99", "3"
	proc, ok := parseProcess("7 (pb worker (test)) "+strings.Join(fields, " "), 7)
	if !ok || proc.ticks != 30 || proc.start != 99 || proc.memory != 3*uint64(os.Getpagesize()) {
		t.Fatalf("process=%+v valid=%v", proc, ok)
	}
	for _, input := range []string{"", "7 (pb) S 1", "not a stat"} {
		if _, ok := parseProcess(input, 7); ok {
			t.Fatalf("accepted invalid stat: %q", input)
		}
	}
}

func TestCollectDisk(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	disk := collectDisk(filepath.Join(dir, "not-created", "data"))
	if disk == nil || disk.TotalBytes == 0 || disk.Path != dir || disk.UsedBytes > disk.TotalBytes {
		t.Fatalf("disk=%+v", disk)
	}
}
