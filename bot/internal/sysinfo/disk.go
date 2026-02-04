package sysinfo

import (
	"syscall"
)

// DiskInfo contains disk usage information.
type DiskInfo struct {
	Path         string
	Total        uint64
	Used         uint64
	Available    uint64
	UsagePercent float64
}

// GetDiskInfo returns disk usage for the specified path.
func GetDiskInfo(path string) (*DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	avail := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	info := &DiskInfo{
		Path:      path,
		Total:     total,
		Used:      used,
		Available: avail,
	}

	if total > 0 {
		info.UsagePercent = 100 * float64(used) / float64(total)
	}

	return info, nil
}
