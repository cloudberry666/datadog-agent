// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package inventories

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
)

type schedulerInterface interface {
	TriggerAndResetCollectorTimer(name string, delay time.Duration)
}

// AutoConfigInterface is an interface for the GetLoadedConfigs method of autodiscovery
type AutoConfigInterface interface {
	GetLoadedConfigs() map[string]integration.Config
}

// CollectorInterface is an interface for the GetAllInstanceIDs method of the collector
type CollectorInterface interface {
	GetAllInstanceIDs(checkName string) []check.ID
}

type checkMetadataCacheEntry struct {
	LastUpdated           time.Time
	CheckInstanceMetadata CheckInstanceMetadata
}

var (
	checkMetadata      = make(map[string]*checkMetadataCacheEntry) // by check ID
	checkMetadataMutex = &sync.Mutex{}
	agentMetadata      = make(AgentMetadata)
	agentMetadataMutex = &sync.Mutex{}

	agentStartupTime = timeNow()

	lastGetPayload      = timeNow()
	lastGetPayloadMutex = &sync.Mutex{}

	metadataUpdatedC = make(chan interface{}, 1)
)

var (
	// For testing purposes
	timeNow   = time.Now
	timeSince = time.Since

	// KnownAgentMetadata lists all known agent-metadata keys.  Setting a key
	// not on this list will result in a runtime panic.
	KnownAgentMetadata = map[string]struct{}{
		// cloud_provider is the name of the local cloud provider
		"cloud_provider": {},
		// hostname_source is the source of the hostname property, such as `gce` or `azure`.
		"hostname_source": {},

		// config_apm_dd_url contains the configuration value apm_config.dd_url
		"config_apm_dd_url": {},
		// config_dd_url contains the configuration value dd_url
		"config_dd_url": {},
		// config_logs_dd_url contains the configuration value logs_config.logs_dd_url
		"config_logs_dd_url": {},
		// config_logs_socks5_proxy_address contains the configuration value logs_config.socks5_proxy_address
		"config_logs_socks5_proxy_address": {},
		// config_no_proxy contains the configuration value proxy.no_proxy.  It is an array of strings.
		"config_no_proxy": {},
		// config_process_dd_url contains the configuration value process_config.process_dd_url
		"config_process_dd_url": {},
		// config_proxy_http contains the configuration value proxy.http
		"config_proxy_http": {},
		// config_proxy_https contains the configuration value proxy.https
		"config_proxy_https": {},
	}
)

// SetAgentMetadata updates the agent metadata value in the cache
func SetAgentMetadata(name string, value interface{}) {
	agentMetadataMutex.Lock()
	defer agentMetadataMutex.Unlock()

	if _, found := KnownAgentMetadata[name]; !found {
		panic(fmt.Sprintf("Agent metadata key %s not defined", name))
	}

	if agentMetadata[name] != value {
		agentMetadata[name] = value

		select {
		case metadataUpdatedC <- nil:
		default: // To make sure this call is not blocking
		}
	}
}

// SetCheckMetadata updates a metadata value for one check instance in the cache.
func SetCheckMetadata(checkID, key string, value interface{}) {
	checkMetadataMutex.Lock()
	defer checkMetadataMutex.Unlock()

	entry, found := checkMetadata[checkID]
	if !found {
		entry = &checkMetadataCacheEntry{
			CheckInstanceMetadata: make(CheckInstanceMetadata),
		}
		checkMetadata[checkID] = entry
	}

	if entry.CheckInstanceMetadata[key] != value {
		entry.LastUpdated = timeNow()
		entry.CheckInstanceMetadata[key] = value

		select {
		case metadataUpdatedC <- nil:
		default: // To make sure this call is not blocking
		}
	}
}

func createCheckInstanceMetadata(checkID, configProvider string) *CheckInstanceMetadata {
	const transientFields = 3

	var checkInstanceMetadata CheckInstanceMetadata
	var lastUpdated time.Time

	if entry, found := checkMetadata[checkID]; found {
		checkInstanceMetadata = make(CheckInstanceMetadata, len(entry.CheckInstanceMetadata)+transientFields)
		for k, v := range entry.CheckInstanceMetadata {
			checkInstanceMetadata[k] = v
		}
		lastUpdated = entry.LastUpdated
	} else {
		checkInstanceMetadata = make(CheckInstanceMetadata, transientFields)
		lastUpdated = agentStartupTime
	}

	checkInstanceMetadata["last_updated"] = lastUpdated.UnixNano()
	checkInstanceMetadata["config.hash"] = checkID
	checkInstanceMetadata["config.provider"] = configProvider

	return &checkInstanceMetadata
}

// CreatePayload fills and returns the inventory metadata payload
func CreatePayload(hostname string, ac AutoConfigInterface, coll CollectorInterface) *Payload {
	checkMetadataMutex.Lock()
	defer checkMetadataMutex.Unlock()

	checkMeta := make(CheckMetadata)

	foundInCollector := map[string]struct{}{}
	if ac != nil {
		configs := ac.GetLoadedConfigs()
		for _, config := range configs {
			checkMeta[config.Name] = make([]*CheckInstanceMetadata, 0)
			instanceIDs := coll.GetAllInstanceIDs(config.Name)
			for _, id := range instanceIDs {
				checkInstanceMetadata := createCheckInstanceMetadata(string(id), config.Provider)
				checkMeta[config.Name] = append(checkMeta[config.Name], checkInstanceMetadata)
				foundInCollector[string(id)] = struct{}{}
			}
		}
	}
	// if metadata where added for check not in the collector we still need
	// to add them to the checkMetadata (this happens when using the
	// 'check' command)
	for id := range checkMetadata {
		if _, found := foundInCollector[id]; !found {
			// id should be "check_name:check_hash"
			parts := strings.SplitN(id, ":", 2)
			checkMeta[parts[0]] = append(checkMeta[parts[0]], createCheckInstanceMetadata(id, ""))
		}
	}

	agentMetadataMutex.Lock()
	defer agentMetadataMutex.Unlock()
	// Creating a copy of agentMetadataCache
	agentMeta := make(AgentMetadata)
	for k, v := range agentMetadata {
		agentMeta[k] = v
	}

	return &Payload{
		Hostname:      hostname,
		Timestamp:     timeNow().UnixNano(),
		CheckMetadata: &checkMeta,
		AgentMetadata: &agentMeta,
	}
}

// GetPayload returns a new inventory metadata payload and updates lastGetPayload
func GetPayload(hostname string, ac AutoConfigInterface, coll CollectorInterface) *Payload {
	lastGetPayloadMutex.Lock()
	defer lastGetPayloadMutex.Unlock()
	lastGetPayload = timeNow()

	return CreatePayload(hostname, ac, coll)
}

// StartMetadataUpdatedGoroutine starts a routine that listens to the metadataUpdatedC
// signal to run the collector out of its regular interval.
func StartMetadataUpdatedGoroutine(sc schedulerInterface, minSendInterval time.Duration) error {
	go func() {
		for {
			<-metadataUpdatedC
			lastGetPayloadMutex.Lock()
			delay := minSendInterval - timeSince(lastGetPayload)
			if delay < 0 {
				delay = 0
			}
			sc.TriggerAndResetCollectorTimer("inventories", delay)
			lastGetPayloadMutex.Unlock()
		}
	}()
	return nil
}
