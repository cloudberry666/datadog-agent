// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package containermeta

import (
	"context"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/errors"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/retry"
)

// GlobalStore is a global instance of the containermeta store available for
// usage by consumers. Run() needs to be called before any data collection
// happens.
var GlobalStore *Store

const (
	retryCollectorInterval = 30 * time.Second
	pullCollectorInterval  = 5 * time.Second
)

type subscriber struct {
	name   string
	ch     chan []Event
	filter *Filter
}

// Store contains metadata for container workloads.
type Store struct {
	mu          sync.RWMutex
	store       map[Kind]map[string]Entity
	subscribers []subscriber

	candidates map[string]Collector
	collectors map[string]Collector

	eventCh chan []Event
}

// NewStore creates a new container metadata store. Call Run to start it.
func NewStore() *Store {
	candidates := make(map[string]Collector)
	for id, c := range collectorCatalog {
		candidates[id] = c()
	}

	return &Store{
		store:       make(map[Kind]map[string]Entity),
		subscribers: []subscriber{},

		candidates: candidates,
		collectors: make(map[string]Collector),
		eventCh:    make(chan []Event),
	}
}

// Run starts the container metadata store.
func (s *Store) Run(ctx context.Context) {
	retryTicker := time.NewTicker(retryCollectorInterval)
	pullTicker := time.NewTicker(pullCollectorInterval)
	health := health.RegisterLiveness("containermeta-store")

	// Dummy ctx and cancel func until the first pull starts
	pullCtx, pullCancel := context.WithCancel(ctx)

	log.Info("containermeta store initialized successfully")

	go func() {
		for {
			select {
			case <-health.C:

			case <-pullTicker.C:
				// pullCtx will always be expired at this point
				// if pullTicker has the same duration as
				// pullCtx, so we cancel just as good practice
				pullCancel()

				pullCtx, pullCancel = context.WithTimeout(ctx, pullCollectorInterval)
				s.pull(pullCtx)

			case evs := <-s.eventCh:
				s.handleEvents(evs)

			case <-retryTicker.C:
				s.startCandidates(ctx)

				if len(s.candidates) == 0 {
					retryTicker.Stop()
				}

			case <-ctx.Done():
				retryTicker.Stop()
				pullTicker.Stop()

				err := health.Deregister()
				if err != nil {
					log.Warnf("error de-registering health check: %s", err)
				}

				return
			}
		}
	}()
}

// Subscribe returns a channel where container metadata events will be streamed
// as they happen.
func (s *Store) Subscribe(name string, filter *Filter) chan []Event {
	// this buffer size is an educated guess, as we know the rate of
	// updates, but not how fast these can be streamed out yet. it most
	// likely should be configurable.
	const bufferSize = 100

	// this is a `ch []Event` instead of a `ch Event` to improve
	// throughput, as bursts of events are as likely to occur as isolated
	// events, especially at startup or with collectors that periodically
	// pull changes.
	ch := make(chan []Event, bufferSize)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers = append(s.subscribers, subscriber{
		name:   name,
		ch:     ch,
		filter: filter,
	})

	if len(s.store) > 0 {
		evs := []Event{}

		for kind, entitiesOfKind := range s.store {
			if !filter.MatchKind(kind) {
				continue
			}

			// TODO(juliogreff): implement filtering by source

			for _, entity := range entitiesOfKind {
				evs = append(evs, Event{
					Type:   EventTypeSet,
					Entity: entity,
				})
			}
		}

		ch <- evs
	}

	return ch
}

// Unsubscribe ends a subscription to entity events and closes its channel.
func (s *Store) Unsubscribe(ch chan []Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.subscribers {
		if sub.ch == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			break
		}
	}

	close(ch)
}

// GetContainer returns metadata about a container.
func (s *Store) GetContainer(id string) (Container, error) {
	var c Container

	entity, err := s.getEntityByKind(KindContainer, id)
	if err != nil {
		return c, err
	}

	c = entity.(Container)

	return c, nil
}

// GetKubernetesPod returns metadata about a Kubernetes pod.
func (s *Store) GetKubernetesPod(id string) (KubernetesPod, error) {
	var p KubernetesPod

	entity, err := s.getEntityByKind(KindKubernetesPod, id)
	if err != nil {
		return p, err
	}

	p = entity.(KubernetesPod)

	return p, nil
}

// GetECSTask returns metadata about an ECS task.
func (s *Store) GetECSTask(id string) (ECSTask, error) {
	var t ECSTask

	entity, err := s.getEntityByKind(KindECSTask, id)
	if err != nil {
		return t, err
	}

	t = entity.(ECSTask)

	return t, nil
}

// Notify notifies the store with a slice of events.
func (s *Store) Notify(events []Event) {
	s.eventCh <- events
}

func (s *Store) startCandidates(ctx context.Context) {
	// NOTE: s.candidates is not guarded by a mutex as it's only called by
	// the store itself, and the store runs on a single goroutine
	for id, c := range s.candidates {
		err := c.Start(ctx, s)

		// Leave candidates that returned a retriable error to be
		// re-started in the next tick
		if err != nil && retry.IsErrWillRetry(err) {
			log.Debugf("containermeta collector %q could not start, but will retry. error: %s", id, err)
			continue
		}

		// Store successfully started collectors for future reference
		if err == nil {
			log.Infof("containermeta collector %q started successfully", id)
			s.collectors[id] = c
		} else {
			log.Info("containermeta collector %q could not start. error: %s", id, err)
		}

		// Remove non-retriable and successfully started collectors
		// from the list of candidates so they're not retried in the
		// next tick
		delete(s.candidates, id)
	}
}

func (s *Store) pull(ctx context.Context) {
	// NOTE: s.collectors is not guarded by a mutex as it's only called by
	// the store itself, and the store runs on a single goroutine. If this
	// method is made public in the future, we need to guard it.
	for id, c := range s.collectors {
		// Run each pull in its own separate goroutine to reduce
		// latency and unlock the main goroutine to do other work.
		go func(id string, c Collector) {
			err := c.Pull(ctx)
			if err != nil {
				log.Warnf("error pulling from collector %q: %s", id, err.Error())
			}
		}(id, c)
	}
}

func (s *Store) handleEvents(evs []Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO(juliogreff): store entities per source

	for _, ev := range evs {
		meta := ev.Entity.GetID()

		entitiesOfKind, ok := s.store[meta.Kind]
		if !ok {
			s.store[meta.Kind] = make(map[string]Entity)
			entitiesOfKind = s.store[meta.Kind]
		}

		switch ev.Type {
		case EventTypeSet:
			entitiesOfKind[meta.ID] = ev.Entity
		case EventTypeUnset:
			delete(entitiesOfKind, meta.ID)
		default:
			log.Errorf("cannot handle event of type %d. event dump: %+v", ev)
		}
	}

	for _, sub := range s.subscribers {
		filter := sub.filter
		filteredEvents := make([]Event, 0, len(evs))

		for _, ev := range evs {
			if filter.Match(ev) {
				filteredEvents = append(filteredEvents, ev)
			}
		}

		sub.ch <- filteredEvents

		log.Debugf("sent %d events to subscriber %q", len(filteredEvents), sub.name)
	}
}

func (s *Store) getEntityByKind(kind Kind, id string) (Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entitiesOfKind, ok := s.store[kind]
	if !ok {
		return nil, errors.NewNotFound(id)
	}

	entity, ok := entitiesOfKind[id]
	if !ok {
		return nil, errors.NewNotFound(id)
	}

	return entity, nil
}

func init() {
	GlobalStore = NewStore()
}
