// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build kubelet

// TODO(juliogreff): remove build tags after we remove kubelet collector and
// move constants here

package collectors

import (
	"context"

	"github.com/gobwas/glob"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/containermeta"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	"github.com/DataDog/datadog-agent/pkg/tagger/utils"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	containermetaCollectorName = "containermeta"
)

// ContainerMetaCollector collects tags from the metadata in the containermeta
// store.
type ContainerMetaCollector struct {
	store *containermeta.Store
	out   chan<- []*TagInfo
	stop  chan struct{}

	labelsAsTags      map[string]string
	annotationsAsTags map[string]string
	globLabels        map[string]glob.Glob
	globAnnotations   map[string]glob.Glob
}

// Detect initializes the ContainerMetaCollector.
func (c *ContainerMetaCollector) Detect(ctx context.Context, out chan<- []*TagInfo) (CollectionMode, error) {
	c.out = out
	c.stop = make(chan struct{})

	labelsAsTags := config.Datadog.GetStringMapString("kubernetes_pod_labels_as_tags")
	annotationsAsTags := config.Datadog.GetStringMapString("kubernetes_pod_annotations_as_tags")
	c.labelsAsTags, c.globLabels = utils.InitMetadataAsTags(labelsAsTags)
	c.annotationsAsTags, c.globAnnotations = utils.InitMetadataAsTags(annotationsAsTags)

	return StreamCollection, nil
}

// Stream runs the continuous event watching loop and sends new tags to the
// tagger based on the events sent by the containermeta.
func (c *ContainerMetaCollector) Stream() error {
	const name = "tagger-containermeta"
	health := health.RegisterLiveness(name)
	ch := c.store.Subscribe(name, nil)

	for {
		select {
		case evBundle := <-ch:
			c.processEvents(evBundle)

		case <-health.C:

		case <-c.stop:
			err := health.Deregister()
			if err != nil {
				log.Warnf("error de-registering health check: %s", err)
			}

			c.store.Unsubscribe(ch)

			return nil
		}
	}
}

// Stop shuts down the ContainerMetaCollector.
func (c *ContainerMetaCollector) Stop() error {
	c.stop <- struct{}{}
	return nil
}

// Fetch is a no-op in the ContainerMetaCollector to prevent expensive and
// race-condition prone forcing of pulls from upstream collectors.  Since
// containermeta.Store will eventually own notifying all downstream consumers,
// this codepath should never trigger anyway.
func (c *ContainerMetaCollector) Fetch(ctx context.Context, entity string) ([]string, []string, []string, error) {
	return nil, nil, nil, nil
}

func containermetaFactory() Collector {
	return &ContainerMetaCollector{
		store: containermeta.GlobalStore,
	}
}

func init() {
	// NOTE: ContainerMetaCollector is meant to be used as the single
	// collector, so priority doesn't matter and should be removed entirely
	// after migration is done.

	// NOTE: uncomment this and remove all other collector registrations
	// once ContainerMetaCollector is ready for release.
	registerCollector(containermetaCollectorName, containermetaFactory, NodeRuntime)
}
