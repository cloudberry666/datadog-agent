// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build kubelet

package collectors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/kubelet"
)

func TestParsePods(t *testing.T) {
	dockerEntityID := "container_id://d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f"
	dockerImageID := "docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4"
	dockerContainerStatus := kubelet.Status{
		Containers: []kubelet.ContainerStatus{
			{
				ID:      dockerEntityID,
				Image:   "datadog/docker-dd-agent:latest5",
				ImageID: dockerImageID,
				Name:    "dd-agent",
			},
		},
		AllContainers: []kubelet.ContainerStatus{
			{
				ID:      dockerEntityID,
				Image:   "datadog/docker-dd-agent:latest5",
				ImageID: dockerImageID,
				Name:    "dd-agent",
			},
		},
		Phase: "Running",
	}
	dockerContainerSpec := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "dd-agent",
				Image: "datadog/docker-dd-agent:latest5",
			},
		},
	}
	dockerContainerSpecWithEnv := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "dd-agent",
				Image: "datadog/docker-dd-agent:latest5",
				Env: []kubelet.EnvVar{
					{
						Name:  "DD_ENV",
						Value: "production",
					},
					{
						Name:  "DD_SERVICE",
						Value: "dd-agent",
					},
					{
						Name:  "DD_VERSION",
						Value: "1.1.0",
					},
				},
			},
		},
	}
	dockerContainerSpecWithInterpolatedEnv := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "dd-agent",
				Image: "datadog/docker-dd-agent:latest5",
				Env: []kubelet.EnvVar{
					{
						Name:  "PROD_ENV",
						Value: "production",
					},
					{
						Name:  "MY_ENV",
						Value: "$(PROD_ENV)2",
					},
					{
						Name:  "DD_ENV",
						Value: "$(MY_ENV)",
					},
					{
						Name:  "DD_SERVICE",
						Value: "dd-agent",
					},
					{
						Name:  "DD_VERSION_MAJOR",
						Value: "1",
					},
					{
						Name:  "DD_VERSION_MINOR",
						Value: "2",
					},
					{
						Name:  "DD_VERSION_PATCH",
						Value: "3",
					},
					{
						Name:  "DD_VERSION",
						Value: "$(DD_VERSION_MAJOR).$(DD_VERSION_MINOR).$(DD_VERSION_PATCH)",
					},
				},
			},
		},
	}

	dockerEntityID2 := "container_id://ff242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f"
	dockerTwoContainersStatus := kubelet.Status{
		Containers: []kubelet.ContainerStatus{
			{
				ID:      dockerEntityID,
				Image:   "datadog/docker-dd-agent:latest5",
				ImageID: dockerImageID,
				Name:    "dd-agent",
			},
			{
				ID:      dockerEntityID2,
				Image:   "datadog/docker-filter:latest",
				ImageID: dockerImageID,
				Name:    "filter",
			},
		},
		AllContainers: []kubelet.ContainerStatus{
			{
				ID:      dockerEntityID,
				Image:   "datadog/docker-dd-agent:latest5",
				ImageID: dockerImageID,
				Name:    "dd-agent",
			},
			{
				ID:      dockerEntityID2,
				Image:   "datadog/docker-filter:latest",
				ImageID: dockerImageID,
				Name:    "filter",
			},
		},
		Phase: "Pending",
	}
	dockerTwoContainersSpec := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "dd-agent",
				Image: "datadog/docker-dd-agent:latest5",
			},
			{
				Name:  "filter",
				Image: "datadog/docker-filter:latest",
			},
		},
	}

	dockerEntityIDCassandra := "container_id://6eaa4782de428f5ea639e33a837ed47aa9fa9e6969f8cb23e39ff788a751ce7d"
	dockerContainerStatusCassandra := kubelet.Status{
		Containers: []kubelet.ContainerStatus{
			{
				ID:      dockerEntityIDCassandra,
				Image:   "gcr.io/google-samples/cassandra:v13",
				ImageID: dockerImageID,
				Name:    "cassandra",
			},
		},
		AllContainers: []kubelet.ContainerStatus{
			{
				ID:      dockerEntityIDCassandra,
				Image:   "gcr.io/google-samples/cassandra:v13",
				ImageID: dockerImageID,
				Name:    "cassandra",
			},
		},
		Phase: "Running",
	}
	dockerContainerSpecCassandra := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "cassandra",
				Image: "gcr.io/google-samples/cassandra:v13",
			},
		},
		Volumes: []kubelet.VolumeSpec{
			{
				Name: "cassandra-data",
				PersistentVolumeClaim: &kubelet.PersistentVolumeClaimSpec{
					ClaimName: "cassandra-data-cassandra-0",
				},
			},
		},
	}

	dockerContainerSpecCassandraMultiplePvcs := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "cassandra",
				Image: "gcr.io/google-samples/cassandra:v13",
			},
		},
		Volumes: []kubelet.VolumeSpec{
			{
				Name: "cassandra-data",
				PersistentVolumeClaim: &kubelet.PersistentVolumeClaimSpec{
					ClaimName: "cassandra-data-cassandra-0",
				},
			},
			{
				Name: "another-pvc",
				PersistentVolumeClaim: &kubelet.PersistentVolumeClaimSpec{
					ClaimName: "another-pvc-data-0",
				},
			},
		},
	}

	criEntityID := "container_id://acbe44ff07525934cab9bf7c38c6627d64fd0952d8e6b87535d57092bfa6e9d1"
	criImageID := "sha256:43940c34f24f39bc9a00b4f9dbcab51a3b28952a7c392c119b877fcb48fe65a3"
	criContainerStatus := kubelet.Status{
		Containers: []kubelet.ContainerStatus{
			{
				ID:      criEntityID,
				Image:   "sha256:0f006d265944c984e05200fab1c14ac54163cbcd4e8ae0ba3b35eb46fc559823",
				ImageID: criImageID,
				Name:    "redis-master",
			},
		},
		AllContainers: []kubelet.ContainerStatus{
			{
				ID:      criEntityID,
				Image:   "sha256:0f006d265944c984e05200fab1c14ac54163cbcd4e8ae0ba3b35eb46fc559823",
				ImageID: criImageID,
				Name:    "redis-master",
			},
		},
		Phase: "Running",
	}
	criContainerSpec := kubelet.Spec{
		Containers: []kubelet.ContainerSpec{
			{
				Name:  "redis-master",
				Image: "gcr.io/google_containers/redis:e2e",
			},
		},
	}

	containerStatusEmptyID := kubelet.Status{
		Containers: []kubelet.ContainerStatus{
			{
				ID:    "",
				Image: "sha256:0f006d265944c984e05200fab1c14ac54163cbcd4e8ae0ba3b35eb46fc559823",
				Name:  "redis-master",
			},
		},
		AllContainers: []kubelet.ContainerStatus{
			{
				ID:    "",
				Image: "sha256:0f006d265944c984e05200fab1c14ac54163cbcd4e8ae0ba3b35eb46fc559823",
				Name:  "redis-master",
			},
		},
		Phase: "Running",
	}

	for nb, tc := range []struct {
		skip              bool
		desc              string
		pod               *kubelet.Pod
		labelsAsTags      map[string]string
		annotationsAsTags map[string]string
		expectedInfo      []*TagInfo
	}{
		{
			desc:         "empty pod",
			pod:          &kubelet.Pod{},
			labelsAsTags: map[string]string{},
			expectedInfo: nil,
		},
		{
			desc: "pod + k8s recommended tags",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name:      "dd-agent-rc-qd876",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dd-agent",
						"app.kubernetes.io/instance":   "dd-agent-rc",
						"app.kubernetes.io/version":    "1.1.0",
						"app.kubernetes.io/component":  "dd-agent",
						"app.kubernetes.io/part-of":    "dd",
						"app.kubernetes.io/managed-by": "spinnaker",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_namespace:default",
					"kube_container_name:dd-agent",
					"image_tag:latest5",
					"kube_app_name:dd-agent",
					"kube_app_instance:dd-agent-rc",
					"kube_app_version:1.1.0",
					"kube_app_component:dd-agent",
					"kube_app_part_of:dd",
					"kube_app_managed_by:spinnaker",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"pod_name:dd-agent-rc-qd876",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"display_container_name:dd-agent_dd-agent-rc-qd876",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "daemonset + common tags",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name:      "dd-agent-rc-qd876",
					Namespace: "default",
					Owners: []kubelet.PodOwner{
						{
							Kind: "DaemonSet",
							Name: "dd-agent-rc",
							ID:   "6a76e51c-88d7-11e7-9a0f-42010a8401cc",
						},
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_namespace:default",
					"kube_container_name:dd-agent",
					"kube_daemon_set:dd-agent-rc",
					"image_tag:latest5",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"kube_ownerref_kind:daemonset",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"pod_name:dd-agent-rc-qd876",
					"kube_ownerref_name:dd-agent-rc",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"display_container_name:dd-agent_dd-agent-rc-qd876",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "two containers + pod",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name:      "dd-agent-rc-qd876",
					Namespace: "default",
					UID:       "5e8e05",
					Owners: []kubelet.PodOwner{
						{
							Kind: "DaemonSet",
							Name: "dd-agent-rc",
							ID:   "6a76e51c-88d7-11e7-9a0f-42010a8401cc",
						},
					},
				},
				Status: dockerTwoContainersStatus,
				Spec:   dockerTwoContainersSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{
				{
					Source: "kubelet",
					Entity: "kubernetes_pod_uid://5e8e05",
					LowCardTags: []string{
						"kube_namespace:default",
						"kube_ownerref_kind:daemonset",
						"kube_daemon_set:dd-agent-rc",
						"pod_phase:pending",
					},
					OrchestratorCardTags: []string{
						"pod_name:dd-agent-rc-qd876",
						"kube_ownerref_name:dd-agent-rc",
					},
					HighCardTags: []string{},
					StandardTags: []string{},
				},
				{
					Source: "kubelet",
					Entity: dockerEntityID,
					LowCardTags: []string{
						"kube_namespace:default",
						"kube_container_name:dd-agent",
						"kube_daemon_set:dd-agent-rc",
						"image_tag:latest5",
						"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
						"image_name:datadog/docker-dd-agent",
						"short_image:docker-dd-agent",
						"kube_ownerref_kind:daemonset",
						"pod_phase:pending",
					},
					OrchestratorCardTags: []string{
						"kube_ownerref_name:dd-agent-rc",
						"pod_name:dd-agent-rc-qd876",
					},
					HighCardTags: []string{
						"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
						"display_container_name:dd-agent_dd-agent-rc-qd876",
					},
					StandardTags: []string{},
				},
				{
					Source: "kubelet",
					Entity: dockerEntityID2,
					LowCardTags: []string{
						"kube_namespace:default",
						"kube_container_name:filter",
						"kube_daemon_set:dd-agent-rc",
						"image_tag:latest",
						"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
						"image_name:datadog/docker-filter",
						"short_image:docker-filter",
						"kube_ownerref_kind:daemonset",
						"pod_phase:pending",
					},
					OrchestratorCardTags: []string{
						"kube_ownerref_name:dd-agent-rc",
						"pod_name:dd-agent-rc-qd876",
					},
					HighCardTags: []string{
						"container_id:ff242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
						"display_container_name:filter_dd-agent-rc-qd876",
					},
					StandardTags: []string{},
				},
			},
		},
		{
			desc: "standalone replicaset",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Owners: []kubelet.PodOwner{
						{
							Kind: "ReplicaSet",
							Name: "kubernetes-dashboard",
						},
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"kube_ownerref_kind:replicaset",
					"kube_replica_set:kubernetes-dashboard",
					"image_tag:latest5",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"kube_ownerref_name:kubernetes-dashboard",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "replicaset to daemonset < 1.8",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Owners: []kubelet.PodOwner{
						{
							Kind: "ReplicaSet",
							Name: "frontend-2891696001",
						},
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"kube_deployment:frontend",
					"kube_replica_set:frontend-2891696001",
					"kube_ownerref_kind:replicaset",
					"image_tag:latest5",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"kube_ownerref_name:frontend-2891696001",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "replicaset to daemonset 1.8+",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Owners: []kubelet.PodOwner{
						{
							Kind: "ReplicaSet",
							Name: "front-end-768dd754b7",
						},
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"kube_deployment:front-end",
					"kube_replica_set:front-end-768dd754b7",
					"kube_ownerref_kind:replicaset",
					"image_tag:latest5",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"kube_ownerref_name:front-end-768dd754b7",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "pod labels",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":         "kube-proxy",
						"tier":              "node",
						"k8s-app":           "kubernetes-dashboard",
						"pod-template-hash": "490794276",
						"GitCommit":         "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam":         "Kenafeh",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
				"tier":      "tier",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "pod labels + annotations",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":         "kube-proxy",
						"tier":              "node",
						"k8s-app":           "kubernetes-dashboard",
						"pod-template-hash": "490794276",
					},
					Annotations: map[string]string{
						"noTag":                          "don't collect",
						"GitCommit":                      "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam":                      "Kenafeh",
						"ad.datadoghq.com/tags":          `{"pod_template_version": "1.0.0"}`,
						"ad.datadoghq.com/dd-agent.tags": `{"agent_version": "6.9.0"}`,
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"tier":      "tier",
			},
			annotationsAsTags: map[string]string{
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_template_version:1.0.0",
					"agent_version:6.9.0",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "standard pod labels",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":                  "kube-proxy",
						"tier":                       "node",
						"k8s-app":                    "kubernetes-dashboard",
						"pod-template-hash":          "490794276",
						"tags.datadoghq.com/env":     "production",
						"tags.datadoghq.com/service": "dd-agent",
						"tags.datadoghq.com/version": "1.1.0",
					},
					Annotations: map[string]string{
						"noTag":     "don't collect",
						"GitCommit": "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam": "Kenafeh",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"tier":      "tier",
			},
			annotationsAsTags: map[string]string{
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"env:production",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"service:dd-agent",
					"version:1.1.0",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{
					"env:production",
					"service:dd-agent",
					"version:1.1.0",
				},
			}},
		},
		{
			desc: "standard container labels",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":                           "kube-proxy",
						"tier":                                "node",
						"k8s-app":                             "kubernetes-dashboard",
						"pod-template-hash":                   "490794276",
						"tags.datadoghq.com/dd-agent.env":     "production",
						"tags.datadoghq.com/dd-agent.service": "dd-agent",
						"tags.datadoghq.com/dd-agent.version": "1.1.0",
					},
					Annotations: map[string]string{
						"noTag":     "don't collect",
						"GitCommit": "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam": "Kenafeh",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"tier":      "tier",
			},
			annotationsAsTags: map[string]string{
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"env:production",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"service:dd-agent",
					"version:1.1.0",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{
					"env:production",
					"service:dd-agent",
					"version:1.1.0",
				},
			}},
		},
		{
			desc: "standard pod + container labels",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":                           "kube-proxy",
						"tier":                                "node",
						"k8s-app":                             "kubernetes-dashboard",
						"pod-template-hash":                   "490794276",
						"tags.datadoghq.com/env":              "production",
						"tags.datadoghq.com/service":          "pod-service",
						"tags.datadoghq.com/version":          "1.2.0",
						"tags.datadoghq.com/dd-agent.env":     "production",
						"tags.datadoghq.com/dd-agent.service": "dd-agent",
						"tags.datadoghq.com/dd-agent.version": "1.1.0",
					},
					Annotations: map[string]string{
						"noTag":     "don't collect",
						"GitCommit": "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam": "Kenafeh",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"tier":      "tier",
			},
			annotationsAsTags: map[string]string{
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"env:production",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"service:dd-agent",
					"service:pod-service",
					"version:1.1.0",
					"version:1.2.0",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{
					"env:production",
					"service:dd-agent",
					"service:pod-service",
					"version:1.1.0",
					"version:1.2.0",
				},
			}},
		},
		{
			desc: "standard container env vars",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":         "kube-proxy",
						"tier":              "node",
						"k8s-app":           "kubernetes-dashboard",
						"pod-template-hash": "490794276",
					},
					Annotations: map[string]string{
						"noTag":     "don't collect",
						"GitCommit": "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam": "Kenafeh",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpecWithEnv,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"tier":      "tier",
			},
			annotationsAsTags: map[string]string{
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"env:production",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"service:dd-agent",
					"version:1.1.0",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{
					"env:production",
					"service:dd-agent",
					"version:1.1.0",
				},
			}},
		},
		{
			desc: "standard container env vars with interpolation",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":         "kube-proxy",
						"tier":              "node",
						"k8s-app":           "kubernetes-dashboard",
						"pod-template-hash": "490794276",
					},
					Annotations: map[string]string{
						"noTag":     "don't collect",
						"GitCommit": "ea38b55f07e40b68177111a2bff1e918132fd5fb",
						"OwnerTeam": "Kenafeh",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpecWithInterpolatedEnv,
			},
			labelsAsTags: map[string]string{
				"component": "component",
				"tier":      "tier",
			},
			annotationsAsTags: map[string]string{
				"ownerteam": "team",
				"gitcommit": "+GitCommit",
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"team:Kenafeh",
					"component:kube-proxy",
					"tier:node",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"env:production2",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"service:dd-agent",
					"version:1.2.3",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"GitCommit:ea38b55f07e40b68177111a2bff1e918132fd5fb",
				},
				StandardTags: []string{
					"env:production2",
					"service:dd-agent",
					"version:1.2.3",
				},
			}},
		},
		{
			desc: "openshift deploymentconfig",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Annotations: map[string]string{
						"openshift.io/deployment-config.latest-version": "1",
						"openshift.io/deployment-config.name":           "gitlab-ce",
						"openshift.io/deployment.name":                  "gitlab-ce-1",
					},
				},
				Status: dockerContainerStatus,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"kube_container_name:dd-agent",
					"oshift_deployment_config:gitlab-ce",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"oshift_deployment:gitlab-ce-1",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "CRI pod",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name: "redis-master-bpnn6",
					Owners: []kubelet.PodOwner{
						{
							Kind: "ReplicaSet",
							Name: "redis-master-546dc4865f",
						},
					},
				},
				Status: criContainerStatus,
				Spec:   criContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: criEntityID,
				LowCardTags: []string{
					"kube_container_name:redis-master",
					"kube_ownerref_kind:replicaset",
					"kube_deployment:redis-master",
					"kube_replica_set:redis-master-546dc4865f",
					"image_id:sha256:43940c34f24f39bc9a00b4f9dbcab51a3b28952a7c392c119b877fcb48fe65a3",
					"image_name:gcr.io/google_containers/redis",
					"image_tag:e2e",
					"short_image:redis",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"kube_ownerref_name:redis-master-546dc4865f",
					"pod_name:redis-master-bpnn6",
				},
				HighCardTags: []string{
					"display_container_name:redis-master_redis-master-bpnn6",
					"container_id:acbe44ff07525934cab9bf7c38c6627d64fd0952d8e6b87535d57092bfa6e9d1",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "pod labels as tags with wildcards",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Labels: map[string]string{
						"component":                    "kube-proxy",
						"tier":                         "node",
						"k8s-app":                      "kubernetes-dashboard",
						"pod-template-hash":            "490794276",
						"app.kubernetes.io/managed-by": "spinnaker",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			labelsAsTags: map[string]string{
				"*":         "foo_%%label%%",
				"component": "component",
			},
			annotationsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"foo_component:kube-proxy",
					"component:kube-proxy",
					"foo_tier:node",
					"foo_k8s-app:kubernetes-dashboard",
					"foo_pod-template-hash:490794276",
					"foo_app.kubernetes.io/managed-by:spinnaker",
					"kube_app_managed_by:spinnaker",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"image_tag:latest5",
					"kube_container_name:dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags:         []string{"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f"},
				StandardTags:         []string{},
			}},
		}, {
			desc: "cronjob",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name:      "hello-1562187720-xzbzh",
					Namespace: "default",
					Owners: []kubelet.PodOwner{
						{
							Kind: "Job",
							Name: "hello-1562187720",
							ID:   "d0dcc17b-9dd5-11e9-b6f0-42010a840064",
						},
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_namespace:default",
					"kube_ownerref_kind:job",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:datadog/docker-dd-agent",
					"image_tag:latest5",
					"kube_container_name:dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
					"kube_cronjob:hello",
				},
				OrchestratorCardTags: []string{
					"kube_job:hello-1562187720",
					"pod_name:hello-1562187720-xzbzh",
					"kube_ownerref_name:hello-1562187720",
				},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
					"display_container_name:dd-agent_hello-1562187720-xzbzh",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "statefulset",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name:      "cassandra-0",
					Namespace: "default",
					Owners: []kubelet.PodOwner{
						{
							Kind: "StatefulSet",
							Name: "cassandra",
							ID:   "0fa7e650-da09-11e9-b8b8-42010af002dd",
						},
					},
				},
				Status: dockerContainerStatusCassandra,
				Spec:   dockerContainerSpecCassandra,
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityIDCassandra,
				LowCardTags: []string{
					"kube_namespace:default",
					"kube_ownerref_kind:statefulset",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:gcr.io/google-samples/cassandra",
					"image_tag:v13",
					"kube_container_name:cassandra",
					"short_image:cassandra",
					"pod_phase:running",
					"kube_stateful_set:cassandra",
					"persistentvolumeclaim:cassandra-data-cassandra-0",
				},
				OrchestratorCardTags: []string{
					"pod_name:cassandra-0",
					"kube_ownerref_name:cassandra",
				},
				HighCardTags: []string{
					"container_id:6eaa4782de428f5ea639e33a837ed47aa9fa9e6969f8cb23e39ff788a751ce7d",
					"display_container_name:cassandra_cassandra-0",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "statefulset 2 pvcs",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name:      "cassandra-0",
					Namespace: "default",
					Owners: []kubelet.PodOwner{
						{
							Kind: "StatefulSet",
							Name: "cassandra",
							ID:   "0fa7e650-da09-11e9-b8b8-42010af002dd",
						},
					},
				},
				Status: dockerContainerStatusCassandra,
				Spec:   dockerContainerSpecCassandraMultiplePvcs,
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityIDCassandra,
				LowCardTags: []string{
					"kube_namespace:default",
					"kube_ownerref_kind:statefulset",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"image_name:gcr.io/google-samples/cassandra",
					"image_tag:v13",
					"kube_container_name:cassandra",
					"short_image:cassandra",
					"pod_phase:running",
					"kube_stateful_set:cassandra",
					"persistentvolumeclaim:cassandra-data-cassandra-0",
					"persistentvolumeclaim:another-pvc-data-0",
				},
				OrchestratorCardTags: []string{
					"kube_ownerref_name:cassandra",
					"pod_name:cassandra-0",
				},
				HighCardTags: []string{
					"container_id:6eaa4782de428f5ea639e33a837ed47aa9fa9e6969f8cb23e39ff788a751ce7d",
					"display_container_name:cassandra_cassandra-0",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "multi-value tags",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Annotations: map[string]string{
						"ad.datadoghq.com/tags":          `{"pod_template_version": "1.0.0", "team": ["A", "B"]}`,
						"ad.datadoghq.com/dd-agent.tags": `{"agent_version": "6.9.0", "python_version": ["2", "3"]}`,
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"kube_container_name:dd-agent",
					"image_tag:latest5",
					"image_name:datadog/docker-dd-agent",
					"short_image:docker-dd-agent",
					"pod_template_version:1.0.0",
					"team:A",
					"team:B",
					"agent_version:6.9.0",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"python_version:2",
					"python_version:3",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags: []string{
					"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f",
				},
				StandardTags: []string{},
			}},
		},
		{
			desc: "pod annotations as tags with wildcards",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Annotations: map[string]string{
						"component":                    "kube-proxy",
						"tier":                         "node",
						"k8s-app":                      "kubernetes-dashboard",
						"pod-template-hash":            "490794276",
						"app.kubernetes.io/managed-by": "spinnaker",
					},
				},
				Status: dockerContainerStatus,
				Spec:   dockerContainerSpec,
			},
			annotationsAsTags: map[string]string{
				"*":         "foo_%%annotation%%",
				"component": "component",
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: dockerEntityID,
				LowCardTags: []string{
					"foo_component:kube-proxy",
					"component:kube-proxy",
					"foo_tier:node",
					"image_id:docker://sha256:77e1fa12f59b01e7e23de95ae01aacc6f09027575ec23b340bb2d6004945f8d4",
					"foo_k8s-app:kubernetes-dashboard",
					"foo_pod-template-hash:490794276",
					"foo_app.kubernetes.io/managed-by:spinnaker",
					"image_name:datadog/docker-dd-agent",
					"image_tag:latest5",
					"kube_container_name:dd-agent",
					"short_image:docker-dd-agent",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{},
				HighCardTags:         []string{"container_id:d0242fc32d53137526dc365e7c86ef43b5f50b6f72dfd53dcb948eff4560376f"},
				StandardTags:         []string{},
			}},
		},
		{
			desc: "Empty container ID",
			pod: &kubelet.Pod{
				Metadata: kubelet.PodMetadata{
					Name: "foo-pod",
					UID:  "foo-uid",
					Owners: []kubelet.PodOwner{
						{
							Kind: "ReplicaSet",
							Name: "foo-rs",
						},
					},
				},
				Status: containerStatusEmptyID,
				Spec:   criContainerSpec,
			},
			labelsAsTags: map[string]string{},
			expectedInfo: []*TagInfo{{
				Source: "kubelet",
				Entity: "kubernetes_pod_uid://foo-uid",
				LowCardTags: []string{
					"kube_ownerref_kind:replicaset",
					"kube_replica_set:foo-rs",
					"pod_phase:running",
				},
				OrchestratorCardTags: []string{
					"kube_ownerref_name:foo-rs",
					"pod_name:foo-pod",
				},
				HighCardTags: []string{},
				StandardTags: []string{},
			}},
		},
	} {
		t.Run(fmt.Sprintf("case %d: %s", nb, tc.desc), func(t *testing.T) {
			if tc.skip {
				t.SkipNow()
			}
			collector := &KubeletCollector{}
			collector.init(nil, nil, tc.labelsAsTags, tc.annotationsAsTags)
			infos, err := collector.parsePods([]*kubelet.Pod{tc.pod})
			assert.Nil(t, err)

			if tc.expectedInfo == nil {
				assert.Len(t, infos, 0)
			} else {
				assertTagInfoListEqual(t, tc.expectedInfo, infos)
			}
		})
	}
}

func Test_parseJSONValue(t *testing.T) {
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
