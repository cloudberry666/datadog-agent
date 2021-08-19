// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package cgroups

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

const (
	sampleCpuShares = "200"
	sampleCpuQuota  = "-1"
	sampleCpuPeriod = "100000"
	sampleCpuStat   = `nr_periods 421
nr_throttled 0
throttled_time 0`
	sameplCpuacctUsage = "115456978426898"
	sameplCpuacctStat  = `user 8718602
system 1439790`
	sampleCpusetCPUs = `0-7`
)

func createCgroupV1FakeCPUFiles(cfs *cgroupMemoryFS, cg *cgroupV1) {
	cfs.setCgroupV1File(cg, "cpu", "cpu.stat", sampleCpuStat)
	cfs.setCgroupV1File(cg, "cpu", "cpu.shares", sampleCpuShares)
	cfs.setCgroupV1File(cg, "cpu", "cpu.cfs_period_us", sampleCpuPeriod)
	cfs.setCgroupV1File(cg, "cpu", "cpu.cfs_quota_us", sampleCpuQuota)
	cfs.setCgroupV1File(cg, "cpuacct", "cpuacct.usage", sameplCpuacctUsage)
	cfs.setCgroupV1File(cg, "cpuacct", "cpuacct.stat", sameplCpuacctStat)
	cfs.setCgroupV1File(cg, "cpuset", "cpuset.cpus", sampleCpusetCPUs)
}

func TestCgroupV1CPUStats(t *testing.T) {
	cfs := newCgroupMemoryFS("/test/fs/cgroup")
	cfs.enableControllers("cpu")

	var err error
	stats := &CPUStats{}

	// Test failure if controller missing (cpuacct is missing)
	tr.reset()
	cgFoo1 := cfs.createCgroupV1("foo1", containerCgroupKubePod(false))
	err = cgFoo1.GetCPUStats(stats)
	assert.ErrorIs(t, err, &ControllerNotFoundError{Controller: "cpuacct"})

	// Test reading files in CPU controllers, all files missing
	tr.reset()
	cfs.enableControllers("cpuacct", "cpuset")
	err = cgFoo1.GetCPUStats(stats)
	assert.NoError(t, err)
	assert.Equal(t, 7, len(tr.errors))
	assert.Equal(t, "", cmp.Diff(CPUStats{}, *stats))

	// Test reading files in CPU controllers, all files present
	tr.reset()
	createCgroupV1FakeCPUFiles(cfs, cgFoo1)
	err = cgFoo1.GetCPUStats(stats)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []error{}, tr.errors)
	assert.Equal(t, "", cmp.Diff(CPUStats{
		User:             UInt64Ptr(8718602 * UserHZToNano),
		System:           UInt64Ptr(1439790 * UserHZToNano),
		Total:            UInt64Ptr(115456978426898),
		Shares:           UInt64Ptr(200),
		ElapsedPeriods:   UInt64Ptr(421),
		ThrottledPeriods: UInt64Ptr(0),
		ThrottledTime:    UInt64Ptr(0),
		SchedulerPeriod:  UInt64Ptr(100000 * uint64(time.Microsecond)),
		SchedulerQuota:   nil,
		CPUCount:         UInt64Ptr(8),
	}, *stats))

	// Test reading files in CPU controllers, all files present except 1 (cpu.shares)
	tr.reset()
	cfs.deleteCgroupV1File(cgFoo1, "cpu", "cpu.shares")
	stats = &CPUStats{}
	err = cgFoo1.GetCPUStats(stats)
	assert.NoError(t, err)
	assert.Equal(t, len(tr.errors), 1)
	assert.Equal(t, "", cmp.Diff(CPUStats{
		User:             UInt64Ptr(8718602 * UserHZToNano),
		System:           UInt64Ptr(1439790 * UserHZToNano),
		Total:            UInt64Ptr(115456978426898),
		Shares:           nil,
		ElapsedPeriods:   UInt64Ptr(421),
		ThrottledPeriods: UInt64Ptr(0),
		ThrottledTime:    UInt64Ptr(0),
		SchedulerPeriod:  UInt64Ptr(100000 * uint64(time.Microsecond)),
		SchedulerQuota:   nil,
		CPUCount:         UInt64Ptr(8),
	}, *stats))
}
