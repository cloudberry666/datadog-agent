// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package cgroups

// PSIStats represent the Pressure Stall Information data
// Source:
// cgroupv1: not present
// cgroupv2: *.pressure
type PSIStats struct {
	Avg10  *float64 // Percentage (0-100)
	Avg60  *float64 // Percentage (0-100)
	Avg300 *float64 // Percentage (0-100)
	Total  *uint64  // Nanoseconds
}

// MemoryStats - all metrics in bytes except if otherwise specified
// All statistics are hierarchical (i.e. includes all descendants)
// Source:
// cgroupv1: memory controller
// cgroupv2: memory controller
type MemoryStats struct {
	UsageTotal   *uint64
	Cache        *uint64
	Swap         *uint64
	RSS          *uint64
	RSSHuge      *uint64
	MappedFile   *uint64
	Pgpgin       *uint64 // Number (no unit), cgroupv1 only
	Pgpgout      *uint64 // Number (no unit), cgroupv1 only
	Pgfault      *uint64 // Number (no unit)
	Pgmajfault   *uint64 // Number (no unit)
	InactiveAnon *uint64
	ActiveAnon   *uint64
	InactiveFile *uint64
	ActiveFile   *uint64
	Unevictable  *uint64

	KernelMemory *uint64

	// This field is mapped to `memory.failcnt` for cgroupv1 and to "oom" in `memory.event`, it does not mean an OOMKill event happened.
	OOMEvents     *uint64 // Number (no unit).
	OOMKiilEvents *uint64 // cgroupv2 only

	Limit             *uint64
	MinThreshold      *uint64 // cgroupv2 only
	LowThreshold      *uint64 // cgroupv1: mapped to soft_limit
	HighThreshold     *uint64 // cgroupv2 only
	SwapLimit         *uint64
	SwapHighThreshold *uint64 // cgroupv2 only

	PSISome PSIStats
	PSIFull PSIStats
}

// CPUStats - all metrics are in nanoseconds execept if otherwise specified
// cgroupv1: cpu/cpuacct/cpuset
type CPUStats struct {
	User   *uint64
	System *uint64
	Total  *uint64

	Shares           *uint64 // Raw share value (no unit)
	ElapsedPeriods   *uint64 // Number (no unit)
	ThrottledPeriods *uint64 // Number (no unit)
	ThrottledTime    *uint64
	CPUCount         *uint64 // Number of accessible logical CPU (from cpuset.cpus)

	SchedulerPeriod *uint64
	SchedulerQuota  *uint64

	PSISome PSIStats
}

type PIDStats struct {
	PIDs                    []int
	HierarchicalThreadCount *uint64 // Number of threads in cgroups + all children
	HierarchicalThreadLimit *uint64 // Maximum number of threads in cgroups + all children
}

type DeviceIOStats struct {
	ReadBytes       *uint64
	WriteBytes      *uint64
	ReadOperations  *uint64
	WriteOperations *uint64

	ReadBytesLimit       *uint64 // cgroupv2 only (bytes/s)
	WriteBytesLimit      *uint64 // cgroupv2 only (bytes/s)
	ReadOperationsLimit  *uint64 // cgroupv2 only (ops/s)
	WriteOperationsLimit *uint64 // cgroupv2 only (ops/s)
}

// IOStats store I/O statistics about a cgroup. Devices identifier in map is MAJOR:MINOR
// cgroupv1: blkio
// cgroupv2: io
type IOStats struct {
	ReadBytes       *uint64
	WriteBytes      *uint64
	ReadOperations  *uint64
	WriteOperations *uint64
	Devices         map[string]DeviceIOStats

	PSISome PSIStats
	PSIFull PSIStats
}

// ContainerMetrics wraps all container metrics
type Stats struct {
	CPU    *CPUStats
	Memory *MemoryStats
	IO     *IOStats
	PID    *PIDStats
}
