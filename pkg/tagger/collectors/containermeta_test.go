// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package collectors

import (
	"fmt"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/containermeta"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
)

func TestHandleKubePod(t *testing.T) {
	const (
		podName      = "datadog-agent-foobar"
		podNamespace = "default"
		env          = "production"
		svc          = "datadog-agent"
		version      = "7.32.0"
	)

	standardTags := []string{
		fmt.Sprintf("env:%s", env),
		fmt.Sprintf("service:%s", svc),
		fmt.Sprintf("version:%s", version),
	}

	podEntityID := containermeta.EntityID{
		Kind: containermeta.KindKubernetesPod,
		ID:   "foobar",
	}

	entityID := fmt.Sprintf("kubernetes_pod_uid://foobar")

	tests := []struct {
		name              string
		labelsAsTags      map[string]string
		annotationsAsTags map[string]string
		pod               containermeta.KubernetesPod
		expected          []*TagInfo
	}{
		{
			name: "fully formed pod (no containers)",
			annotationsAsTags: map[string]string{
				"gitcommit": "+gitcommit",
				"component": "component",
			},
			labelsAsTags: map[string]string{
				"ownerteam": "team",
				"tier":      "tier",
			},
			pod: containermeta.KubernetesPod{
				EntityID: podEntityID,
				EntityMeta: containermeta.EntityMeta{
					Name:      podName,
					Namespace: podNamespace,
					Annotations: map[string]string{
						// Annotations as tags
						"GitCommit": "foobar",
						"ignoreme":  "ignore",
						"component": "agent",

						// Custom tags from map
						"ad.datadoghq.com/tags": `{"pod_template_version":"1.0.0"}`,
					},
					Labels: map[string]string{
						// Labels as tags
						"OwnerTeam":         "container-integrations",
						"tier":              "node",
						"pod-template-hash": "490794276",

						// Standard tags
						"tags.datadoghq.com/env":     env,
						"tags.datadoghq.com/service": svc,
						"tags.datadoghq.com/version": version,

						// K8s recommended tags
						"app.kubernetes.io/name":       svc,
						"app.kubernetes.io/instance":   podName,
						"app.kubernetes.io/version":    version,
						"app.kubernetes.io/component":  "agent",
						"app.kubernetes.io/part-of":    "datadog",
						"app.kubernetes.io/managed-by": "helm",
					},
				},

				// Owner tags
				Owners: []containermeta.KubernetesPodOwner{
					{
						Kind: kubernetes.DeploymentKind,
						Name: svc,
					},
				},

				// PVC tags
				PersistentVolumeClaimNames: []string{"pvc-0"},

				// Phase tags
				Phase: "Running",

				// Container tags
				Containers: []string{},
			},
			expected: []*TagInfo{
				{
					Source: containermetaCollectorName,
					Entity: entityID,
					HighCardTags: []string{
						"gitcommit:foobar",
					},
					OrchestratorCardTags: []string{
						fmt.Sprintf("pod_name:%s", podName),
						"kube_ownerref_name:datadog-agent",
					},
					LowCardTags: append([]string{
						fmt.Sprintf("kube_app_instance:%s", podName),
						fmt.Sprintf("kube_app_name:%s", svc),
						fmt.Sprintf("kube_app_version:%s", version),
						fmt.Sprintf("kube_deployment:%s", svc),
						fmt.Sprintf("kube_namespace:%s", podNamespace),
						"component:agent",
						"kube_app_component:agent",
						"kube_app_managed_by:helm",
						"kube_app_part_of:datadog",
						"kube_ownerref_kind:deployment",
						"pod_phase:running",
						"pod_template_version:1.0.0",
						"team:container-integrations",
						"tier:node",
					}, standardTags...),
					StandardTags: standardTags,
				},
			},
		},
		{
			name: "pod with containers",
			pod: containermeta.KubernetesPod{
				EntityID: podEntityID,
				EntityMeta: containermeta.EntityMeta{
					Name:      podName,
					Namespace: podNamespace,
				},
				// TODO(juliogreff): add containers
				Containers: []string{},
			},
			expected: []*TagInfo{
				{
					Source:       containermetaCollectorName,
					Entity:       entityID,
					HighCardTags: []string{},
					OrchestratorCardTags: []string{
						fmt.Sprintf("pod_name:%s", podName),
					},
					LowCardTags: append([]string{
						fmt.Sprintf("kube_namespace:%s", podNamespace),
					}),
					StandardTags: []string{},
				},
			},
		},
		{
			name: "pod from openshift deployment",
			pod: containermeta.KubernetesPod{
				EntityID: podEntityID,
				EntityMeta: containermeta.EntityMeta{
					Name:      podName,
					Namespace: podNamespace,
					Annotations: map[string]string{
						"openshift.io/deployment-config.latest-version": "1",
						"openshift.io/deployment-config.name":           "gitlab-ce",
						"openshift.io/deployment.name":                  "gitlab-ce-1",
					},
				},
			},
			expected: []*TagInfo{
				{
					Source:       containermetaCollectorName,
					Entity:       entityID,
					HighCardTags: []string{},
					OrchestratorCardTags: []string{
						fmt.Sprintf("pod_name:%s", podName),
						"oshift_deployment:gitlab-ce-1",
					},
					LowCardTags: append([]string{
						fmt.Sprintf("kube_namespace:%s", podNamespace),
						"oshift_deployment_config:gitlab-ce",
					}),
					StandardTags: []string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := &ContainerMetaCollector{}
			collector.init(tt.labelsAsTags, tt.annotationsAsTags)

			actual := collector.handleKubePod(containermeta.Event{
				Type:   containermeta.EventTypeSet,
				Entity: tt.pod,
			})

			assertTagInfoListEqual(t, tt.expected, actual)
		})
	}
}

func TestParseJSONValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    map[string][]string
		wantErr bool
	}{
		{
			name:    "empty json",
			value:   ``,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid json",
			value:   `{key}`,
			want:    nil,
			wantErr: true,
		},
		{
			name:  "invalid value",
			value: `{"key1": "val1", "key2": 0}`,
			want: map[string][]string{
				"key1": {"val1"},
			},
			wantErr: false,
		},
		{
			name:  "strings and arrays",
			value: `{"key1": "val1", "key2": ["val2"]}`,
			want: map[string][]string{
				"key1": {"val1"},
				"key2": {"val2"},
			},
			wantErr: false,
		},
		{
			name:  "arrays only",
			value: `{"key1": ["val1", "val11"], "key2": ["val2", "val22"]}`,
			want: map[string][]string{
				"key1": {"val1", "val11"},
				"key2": {"val2", "val22"},
			},
			wantErr: false,
		},
		{
			name:  "strings only",
			value: `{"key1": "val1", "key2": "val2"}`,
			want: map[string][]string{
				"key1": {"val1"},
				"key2": {"val2"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJSONValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Len(t, got, len(tt.want))
			for k, v := range tt.want {
				assert.ElementsMatch(t, v, got[k])
			}
		})
	}
}
