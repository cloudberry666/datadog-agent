package listeners

import (
	"testing"

	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/containermeta"
	"github.com/stretchr/testify/assert"
)

func TestCreateKubeletServices(t *testing.T) {
	config.Datadog.SetDefault("ac_include", []string{"name:baz"})
	config.Datadog.SetDefault("ac_exclude", []string{"image:datadoghq.com/baz.*"})
	config.Datadog.SetDefault("container_exclude_metrics", []string{"name:metrics-excluded"})
	config.Datadog.SetDefault("container_exclude_logs", []string{"name:logs-excluded"})
	config.Datadog.SetDefault("exclude_pause_container", true)

	defer func() {
		config.Datadog.SetDefault("ac_include", []string{})
		config.Datadog.SetDefault("ac_exclude", []string{})
		config.Datadog.SetDefault("container_exclude_metrics", []string{})
		config.Datadog.SetDefault("container_exclude_logs", []string{})
		config.Datadog.SetDefault("exclude_pause_container", true)
	}()

	const (
		containerID   = "foobarquux"
		containerName = "agent"
		podID         = "foobar"
		podName       = "datadog-agent-foobar"
		podNamespace  = "default"
		env           = "production"
		svc           = "datadog-agent"
		version       = "7.32.0"
	)

	pod := containermeta.KubernetesPod{
		EntityID: containermeta.EntityID{
			Kind: containermeta.KindKubernetesPod,
			ID:   podID,
		},
		EntityMeta: containermeta.EntityMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Containers: []string{containerID},
	}

	container := containermeta.Container{
		EntityID: containermeta.EntityID{
			Kind: containermeta.KindContainer,
			ID:   containerID,
		},
		EntityMeta: containermeta.EntityMeta{
			Name: containerName,
		},
	}

	newSvcCh := make(chan Service)
	doneCh := make(chan struct{})
	actualServices := make(map[string]Service)
	expectedServices := map[string]Service{
		"kubernetes_pod://foobar": &KubePodService{
			entity:        "kubernetes_pod://foobar",
			adIdentifiers: []string{"kubernetes_pod://foobar"},
			hosts: map[string]string{
				"pod": "",
			},
			creationTime: integration.After,
		},
		"container_id://foobarquux": &KubeContainerService{
			entity:        "container_id://foobarquux",
			adIdentifiers: []string{"container_id://foobarquux", ""},
			hosts: map[string]string{
				"pod": "",
			},
			ports:           []ContainerPort{},
			creationTime:    integration.After,
			ready:           false,
			checkNames:      nil,
			metricsExcluded: true,
			logsExcluded:    false,
			extraConfig: map[string]string{
				"namespace": podNamespace,
				"pod_name":  podName,
				"pod_uid":   podID,
			},
		},
	}

	go func() {
		for svc := range newSvcCh {
			if svc == nil {
				break
			}

			actualServices[svc.GetEntity()] = svc
		}

		close(doneCh)
	}()

	filters, err := newContainerFilters()
	if err != nil {
		t.Fatalf("cannot initialize container filters: %s", err)
	}

	listener := &KubeletContainerMetaListener{
		services:   make(map[string]Service),
		newService: newSvcCh,
		filters:    filters,
	}

	listener.createContainerService(pod, container, false)
	listener.createPodService(pod, []containermeta.Container{container}, false)

	close(newSvcCh)

	<-doneCh

	for entity, expectedSvc := range expectedServices {
		actualSvc, ok := actualServices[entity]
		if !ok {
			t.Errorf("expected to find service %q, but it was not generated", entity)
		}

		assert.Equal(t, expectedSvc, actualSvc)

		delete(actualServices, entity)
	}

	if len(actualServices) > 0 {
		t.Errorf("got unexpected services: %+v", actualServices)
	}
}
