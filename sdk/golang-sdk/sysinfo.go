package daedalus

import (
	"fmt"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// GetSystemInfo collects current system metrics (CPU, memory, disk, OS, hostname).
func GetSystemInfo() map[string]string {
	info := make(map[string]string)

	// CPU usage
	percentages, err := cpu.Percent(0, false)
	if err == nil && len(percentages) > 0 {
		info["CPU"] = fmt.Sprintf("%.2f", percentages[0])
	} else {
		info["CPU"] = "N/A"
	}

	// Memory usage
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		info["Memory"] = fmt.Sprintf("%.2f", vmStat.UsedPercent)
	} else {
		info["Memory"] = "N/A"
	}

	// Disk usage
	diskStat, err := disk.Usage("/")
	if err == nil {
		info["Disk"] = fmt.Sprintf("%.2f", diskStat.UsedPercent)
	} else {
		info["Disk"] = "N/A"
	}

	// OS
	info["OS"] = runtime.GOOS

	// Hostname
	hostname, err := os.Hostname()
	if err == nil {
		info["Hostname"] = hostname
	} else {
		info["Hostname"] = "unknown"
	}

	return info
}
