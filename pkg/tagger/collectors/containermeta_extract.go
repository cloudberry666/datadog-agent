// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build kubelet

// TODO(juliogreff): remove build tags after we remove kubelet collector and
// move constants here

package collectors

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/containermeta"
	"github.com/DataDog/datadog-agent/pkg/tagger/utils"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/kubelet"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func (c *ContainerMetaCollector) processEvents(evs []containermeta.Event) {
	tagInfos := []*TagInfo{}

	for _, ev := range evs {
		entity := ev.Entity
		entityID := entity.GetID()

		switch ev.Type {
		case containermeta.EventTypeSet:
			switch entityID.Kind {
			case containermeta.KindContainer:
				// TODO
			case containermeta.KindKubernetesPod:
				tagInfos = append(tagInfos, c.handleKubePod(ev)...)
			case containermeta.KindECSTask:
				// TODO
			default:
				log.Errorf("cannot handle event for entity %q with kind %q", entityID.ID, entityID.Kind)
			}

		case containermeta.EventTypeUnset:
			tagInfos = append(tagInfos, &TagInfo{
				Source:       containermetaCollectorName,
				Entity:       buildTaggerEntityID(entityID),
				DeleteEntity: true,
			})

		default:
			log.Errorf("cannot handle event of type %d", ev.Type)
		}

	}

	c.out <- tagInfos
}

func (c *ContainerMetaCollector) handleKubePod(ev containermeta.Event) []*TagInfo {
	tagInfos := []*TagInfo{}
	tags := utils.NewTagList()

	pod := ev.Entity.(containermeta.KubernetesPod)

	tags.AddOrchestrator(kubernetes.PodTagName, pod.Name)
	tags.AddLow(kubernetes.NamespaceTagName, pod.Namespace)
	tags.AddLow("pod_phase", strings.ToLower(pod.Phase))

	for name, value := range pod.Labels {
		switch name {
		case kubernetes.EnvTagLabelKey:
			tags.AddStandard(tagKeyEnv, value)
		case kubernetes.VersionTagLabelKey:
			tags.AddStandard(tagKeyVersion, value)
		case kubernetes.ServiceTagLabelKey:
			tags.AddStandard(tagKeyService, value)
		case kubernetes.KubeAppNameLabelKey:
			tags.AddLow(tagKeyKubeAppName, value)
		case kubernetes.KubeAppInstanceLabelKey:
			tags.AddLow(tagKeyKubeAppInstance, value)
		case kubernetes.KubeAppVersionLabelKey:
			tags.AddLow(tagKeyKubeAppVersion, value)
		case kubernetes.KubeAppComponentLabelKey:
			tags.AddLow(tagKeyKubeAppComponent, value)
		case kubernetes.KubeAppPartOfLabelKey:
			tags.AddLow(tagKeyKubeAppPartOf, value)
		case kubernetes.KubeAppManagedByLabelKey:
			tags.AddLow(tagKeyKubeAppManagedBy, value)
		}

		utils.AddMetadataAsTags(name, value, c.labelsAsTags, c.globLabels, tags)
	}

	for name, value := range pod.Annotations {
		utils.AddMetadataAsTags(name, value, c.annotationsAsTags, c.globAnnotations, tags)
	}

	if podTags, found := extractTagsFromMap(podTagsAnnotation, pod.Annotations); found {
		for tagName, values := range podTags {
			for _, val := range values {
				tags.AddAuto(tagName, val)
			}
		}
	}

	// OpenShift pod annotations
	if dcName, found := pod.Annotations["openshift.io/deployment-config.name"]; found {
		tags.AddLow("oshift_deployment_config", dcName)
	}
	if deployName, found := pod.Annotations["openshift.io/deployment.name"]; found {
		tags.AddOrchestrator("oshift_deployment", deployName)
	}

	for _, owner := range pod.Owners {
		tags.AddLow(kubernetes.OwnerRefKindTagName, strings.ToLower(owner.Kind))
		tags.AddOrchestrator(kubernetes.OwnerRefNameTagName, owner.Name)

		switch owner.Kind {
		case kubernetes.DeploymentKind:
			tags.AddLow(kubernetes.DeploymentTagName, owner.Name)

		case kubernetes.DaemonSetKind:
			tags.AddLow(kubernetes.DaemonSetTagName, owner.Name)

		case kubernetes.ReplicationControllerKind:
			tags.AddLow(kubernetes.ReplicationControllerTagName, owner.Name)

		case kubernetes.StatefulSetKind:
			tags.AddLow(kubernetes.StatefulSetTagName, owner.Name)
			for _, pvc := range pod.PersistentVolumeClaimNames {
				if pvc != "" {
					tags.AddLow("persistentvolumeclaim", pvc)
				}
			}

		case kubernetes.JobKind:
			cronjob := kubernetes.ParseCronJobForJob(owner.Name)
			if cronjob != "" {
				tags.AddOrchestrator(kubernetes.JobTagName, owner.Name)
				tags.AddLow(kubernetes.CronJobTagName, cronjob)
			} else {
				tags.AddLow(kubernetes.JobTagName, owner.Name)
			}

		case kubernetes.ReplicaSetKind:
			deployment := kubernetes.ParseDeploymentForReplicaSet(owner.Name)
			if len(deployment) > 0 {
				tags.AddLow(kubernetes.DeploymentTagName, deployment)
			}
			tags.AddLow(kubernetes.ReplicaSetTagName, owner.Name)

		case "":

		default:
			log.Debugf("unknown owner kind %q for pod %q", owner.Kind, pod.Name)
		}
	}

	low, orch, high, standard := tags.Compute()
	tagInfos = append(tagInfos, &TagInfo{
		Source:               containermetaCollectorName,
		Entity:               buildTaggerEntityID(pod.EntityID),
		HighCardTags:         high,
		OrchestratorCardTags: orch,
		LowCardTags:          low,
		StandardTags:         standard,
	})

	for _, containerID := range pod.Containers {
		container, err := c.store.GetContainer(containerID)
		if err != nil {
			log.Debugf("pod %q has reference to non-existing container %q", pod.Name, containerID)
			continue
		}

		cTags := tags.Copy()
		cTags.AddLow("kube_container_name", container.Name)
		cTags.AddLow("image_id", container.Image.ID)
		cTags.AddHigh("container_id", container.ID)
		if container.Name != "" && pod.Name != "" {
			cTags.AddHigh("display_container_name", fmt.Sprintf("%s_%s", container.Name, pod.Name))
		}

		// Enrich with standard tags from labels for this container if present
		labelTags := []string{tagKeyEnv, tagKeyVersion, tagKeyService}
		for _, tag := range labelTags {
			label := fmt.Sprintf(podStandardLabelPrefix+"%s.%s", container.Name, tag)

			if value, ok := pod.Labels[label]; ok {
				cTags.AddStandard(tag, value)
			}
		}

		// container-specific tags provided through pod annotation
		containerTags, found := extractTagsFromMap(
			fmt.Sprintf(podContainerTagsAnnotationFormat, container.Name),
			pod.Annotations,
		)
		if found {
			for tagName, values := range containerTags {
				for _, val := range values {
					cTags.AddAuto(tagName, val)
				}
			}
		}

		standardTagKeys := []string{tagKeyEnv, tagKeyVersion, tagKeyService}
		for _, key := range standardTagKeys {
			value, ok := container.EnvVars[key]
			if !ok || value == "" {
				continue
			}

			cTags.AddStandard(key, value)
		}

		image := container.Image
		cTags.AddLow("image_name", image.Name)
		cTags.AddLow("short_image", image.ShortName)
		cTags.AddLow("image_tag", image.Tag)

		low, orch, high, standard := cTags.Compute()
		tagInfos = append(tagInfos, &TagInfo{
			Source:               containermetaCollectorName,
			Entity:               buildTaggerEntityID(container.EntityID),
			HighCardTags:         high,
			OrchestratorCardTags: orch,
			LowCardTags:          low,
			StandardTags:         standard,
		})
	}

	return tagInfos
}

func buildTaggerEntityID(entityID containermeta.EntityID) string {
	switch entityID.Kind {
	case containermeta.KindContainer:
		return containers.BuildTaggerEntityName(entityID.ID)
	case containermeta.KindKubernetesPod:
		return kubelet.PodUIDToTaggerEntityName(entityID.ID)
	case containermeta.KindECSTask:
		// TODO
	default:
		log.Errorf("can't recognize entity %q with kind %q, but building a a tagger ID anyway", entityID.ID, entityID.Kind)
		return containers.BuildEntityName(string(entityID.Kind), entityID.ID)
	}

	return ""
}
