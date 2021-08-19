// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package cgroups

type Cgroup interface {
	Identifier() string
	GetParent() (Cgroup, error)
	GetStats(*Stats) error
	GetCPUStats(*CPUStats) error
	GetMemoryStats(*MemoryStats) error
	GetIOStats(*IOStats) error
	GetPIDStats(*PIDStats) error
}
