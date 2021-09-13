// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build kubelet

package listeners

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/autodiscovery/common/utils"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/containermeta"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/kubelet"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func init() {
	Register("kubelet", NewKubeletContainerMetaListener)
}

// KubeletContainerMetaListener listens to pod creation through a subscription
// to the containermeta store.
type KubeletContainerMetaListener struct {
	store *containermeta.Store
	stop  chan struct{}

	mu       sync.RWMutex
	filters  *containerFilters
	services map[string]Service

	newService chan<- Service
	delService chan<- Service
}

// NewKubeletContainerMetaListener returns a new KubeletContainerMetaListener.
func NewKubeletContainerMetaListener() (ServiceListener, error) {
	filters, err := newContainerFilters()
	if err != nil {
		return nil, err
	}

	return &KubeletContainerMetaListener{
		store:    containermeta.GetGlobalStore(),
		filters:  filters,
		services: make(map[string]Service),
		stop:     make(chan struct{}),
	}, nil
}

// Listen starts listening to events from the containermeta store.
func (l *KubeletContainerMetaListener) Listen(newSvc chan<- Service, delSvc chan<- Service) {
	l.newService = newSvc
	l.delService = delSvc

	const name = "ad-containermeta-kubeletlistener"

	ch := l.store.Subscribe(name, containermeta.NewFilter(
		[]containermeta.Kind{containermeta.KindKubernetesPod},
		[]string{"kubelet"},
	))
	health := health.RegisterLiveness(name)
	firstRun := true

	go func() {
		for {
			select {
			case evBundle := <-ch:
				l.processEvents(evBundle, firstRun)
				firstRun = false

			case <-health.C:

			case <-l.stop:
				err := health.Deregister()
				if err != nil {
					log.Warnf("error de-registering health check: %s", err)
				}

				l.store.Unsubscribe(ch)

				return
			}
		}
	}()
}

// Stop stops the KubeletContainerMetaListener.
func (l *KubeletContainerMetaListener) Stop() {
	l.stop <- struct{}{}
}

func (l *KubeletContainerMetaListener) processEvents(evBundle containermeta.EventBundle, firstRun bool) {
	for _, ev := range evBundle.Events {
		entity := ev.Entity
		entityID := entity.GetID()

		if entityID.Kind != containermeta.KindKubernetesPod {
			log.Errorf("got event %d with entity of kind %q. filters broken?", ev.Type, entityID.Kind)
		}

		switch ev.Type {
		case containermeta.EventTypeSet:
			pod := entity.(containermeta.KubernetesPod)
			l.processPod(pod, firstRun)

		case containermeta.EventTypeUnset:
			l.removeService(entityID)

		default:
			log.Errorf("cannot handle event of type %d", ev.Type)
		}
	}

	close(evBundle.Ch)
}

func (l *KubeletContainerMetaListener) processPod(pod containermeta.KubernetesPod, firstRun bool) {
	containers := make([]containermeta.Container, 0, len(pod.Containers))

	for _, containerID := range pod.Containers {
		container, err := l.store.GetContainer(containerID)
		if err != nil {
			log.Debugf("pod %q has reference to non-existing container %q", pod.Name, containerID)
			continue
		}

		l.createContainerService(pod, container, firstRun)

		containers = append(containers, container)
	}

	l.createPodService(pod, containers, firstRun)
}

func (l *KubeletContainerMetaListener) createPodService(pod containermeta.KubernetesPod, containers []containermeta.Container, firstRun bool) {
	var crTime integration.CreationTime
	if firstRun {
		crTime = integration.Before
	} else {
		crTime = integration.After
	}

	var ports []ContainerPort
	for _, container := range containers {
		for _, port := range container.Ports {
			ports = append(ports, ContainerPort{
				Port: port.Port,
				Name: port.Name,
			})
		}
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Port < ports[j].Port
	})

	entity := kubelet.PodUIDToEntityName(pod.ID)
	svc := &KubePodService{
		entity:        entity,
		adIdentifiers: []string{entity},
		hosts:         map[string]string{"pod": pod.IP},
		ports:         ports,
		creationTime:  crTime,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.services[buildSvcID(pod.GetID())] = svc
	l.newService <- svc
}

func (l *KubeletContainerMetaListener) createContainerService(pod containermeta.KubernetesPod, container containermeta.Container, firstRun bool) {
	containerImg := container.Image
	if l.filters.IsExcluded(containers.GlobalFilter, container.Name, containerImg.RawName, pod.Namespace) {
		log.Debugf("container %s filtered out: name %q image %q namespace %q", container.ID, container.Name, container.Image.Name, pod.Namespace)
		return
	}

	if !container.State.FinishedAt.IsZero() {
		finishedAt := container.State.FinishedAt
		excludeAge := time.Duration(config.Datadog.GetInt("container_exclude_stopped_age")) * time.Hour
		if time.Now().Sub(finishedAt) > excludeAge {
			log.Debugf("container %q not running for too long, skipping", container.ID)
			return
		}
	}

	var crTime integration.CreationTime
	if firstRun {
		crTime = integration.Before
	} else {
		crTime = integration.After
	}

	ports := []ContainerPort{}
	for _, port := range container.Ports {
		ports = append(ports, ContainerPort{
			Port: port.Port,
			Name: port.Name,
		})
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Port < ports[j].Port
	})

	entity := containers.BuildEntityName("container_id", container.ID)
	svc := &KubeContainerService{
		entity:       entity,
		creationTime: crTime,
		ready:        pod.Ready,
		ports:        ports,
		extraConfig: map[string]string{
			"pod_name":  pod.Name,
			"namespace": pod.Namespace,
			"pod_uid":   pod.ID,
		},
		hosts: map[string]string{"pod": pod.IP},

		// Exclude non-running containers (including init containers)
		// from metrics collection but keep them for collecting logs.
		metricsExcluded: l.filters.IsExcluded(
			containers.MetricsFilter,
			container.Name,
			containerImg.RawName,
			pod.Namespace,
		) || !container.State.Running,
		logsExcluded: l.filters.IsExcluded(
			containers.LogsFilter,
			container.Name,
			containerImg.RawName,
			pod.Namespace,
		),
	}

	adIdentifier := container.Name

	// Check for custom AD identifiers
	if customADID, found := utils.GetCustomCheckID(pod.Annotations, container.Name); found {
		adIdentifier = customADID
		svc.adIdentifiers = append(svc.adIdentifiers, customADID)
	}

	// Add container uid as ID
	svc.adIdentifiers = append(svc.adIdentifiers, entity)

	// Cache check names if the pod template is annotated
	if podHasADTemplate(pod.Annotations, adIdentifier) {
		var err error
		svc.checkNames, err = getCheckNamesFromAnnotations(pod.Annotations, adIdentifier)
		if err != nil {
			log.Error(err.Error())
		}
	}

	svc.adIdentifiers = append(svc.adIdentifiers, containerImg.RawName)

	if len(containerImg.ShortName) > 0 && containerImg.ShortName != containerImg.RawName {
		svc.adIdentifiers = append(svc.adIdentifiers, containerImg.ShortName)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if old, found := l.services[entity]; found {
		if kubeletSvcEqual(old, svc) {
			log.Tracef("Received a duplicated kubelet service '%s'", svc.entity)
			return
		}

		log.Tracef("Kubelet service '%s' has been updated, removing the old one", svc.entity)
		l.delService <- old
	}

	l.services[buildSvcID(container.GetID())] = svc
	l.newService <- svc
}

func (l *KubeletContainerMetaListener) removeService(entityID containermeta.EntityID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	svcID := buildSvcID(entityID)
	svc, ok := l.services[svcID]
	if !ok {
		log.Debugf("service %q not found, not removing", svcID)
		return
	}

	delete(l.services, svcID)
	l.delService <- svc
}

func buildSvcID(entityID containermeta.EntityID) string {
	return fmt.Sprintf("%s://%s", entityID.Kind, entityID.ID)
}
