//go:build !linux

package systemstatus

// Unsupported platforms still report the OS, CPU count and active instances.
// OS-specific measurements remain null rather than reporting misleading zeros.
func collectPlatform(_ string, _ map[string]int) sample { return sample{} }
