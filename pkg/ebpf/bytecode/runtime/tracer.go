// Code generated by go generate; DO NOT EDIT.
// +build linux_bpf

package runtime

import (
	"github.com/DataDog/datadog-agent/pkg/ebpf"
)

var Tracer = ebpf.NewRuntimeAsset("tracer.c", "c5843dc37146036b21ffb3b7cc9bef4e901642e7cb1a1b10fa50883da70f8487")