// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build kubeapiserver,kubelet

package kubemetadata

import (
	"context"
	"errors"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/containermeta"
)

const (
	collectorID = "kube_metadata"
)

type collector struct {
	store *containermeta.Store
}

func init() {
	containermeta.RegisterCollector(collectorID, func() containermeta.Collector {
		return &collector{}
	})
}

func (c *collector) Start(_ context.Context, store *containermeta.Store) error {
	if !config.IsFeaturePresent(config.Kubernetes) {
		return errors.New("the Agent is not running in Kubernetes")
	}

	c.store = store

	return nil
}

func (c *collector) Pull(ctx context.Context) error {
	events := []containermeta.Event{}

	c.store.Notify(events)

	return nil
}
