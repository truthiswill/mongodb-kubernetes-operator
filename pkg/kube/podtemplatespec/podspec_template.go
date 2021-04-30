package podtemplatespec

import (
	"github.com/mongodb/mongodb-kubernetes-operator/pkg/kube/container"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Modification func(*corev1.PodTemplateSpec)

const (
	notFound = -1
)

func New(templateMods ...Modification) corev1.PodTemplateSpec {
	podTemplateSpec := corev1.PodTemplateSpec{}
	for _, templateMod := range templateMods {
		templateMod(&podTemplateSpec)
	}
	return podTemplateSpec
}

// Apply returns a function which applies a series of Modification functions to a *corev1.PodTemplateSpec
func Apply(templateMods ...Modification) Modification {
	return func(template *corev1.PodTemplateSpec) {
		for _, f := range templateMods {
			f(template)
		}
	}
}

// NOOP is a valid Modification which applies no changes
func NOOP() Modification {
	return func(spec *corev1.PodTemplateSpec) {}
}

// WithContainer applies the modifications to the container with the provided name
func WithContainer(name string, containerfunc func(*corev1.Container)) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		idx := findIndexByName(name, podTemplateSpec.Spec.Containers)
		if idx == notFound {
			// if we are attempting to modify a container that does not exist, we will add a new one
			podTemplateSpec.Spec.Containers = append(podTemplateSpec.Spec.Containers, corev1.Container{Name: name})
			idx = len(podTemplateSpec.Spec.Containers) - 1
		}
		c := &podTemplateSpec.Spec.Containers[idx]
		containerfunc(c)
	}
}

// WithContainerFrom copies the container with name `name` into a new one named `newContainerName`
// and applies the modifications to it, then adds it to the containers.
// If no container with that name exists, it returns NOOP
func WithContainerFrom(name string, newContainerName string, funcs ...func(container *corev1.Container)) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		from := FindContainerByName(name, podTemplateSpec)
		if from == nil {
			return
		}
		dest := from.DeepCopy()
		dest.Name = newContainerName
		for _, f := range funcs {
			f(dest)
		}
		podTemplateSpec.Spec.Containers = append(podTemplateSpec.Spec.Containers, *dest)
	}
}

// WithContainerByIndex applies the modifications to the container with the provided index
// if the index is out of range, a new container is added to accept these changes.
func WithContainerByIndex(index int, funcs ...func(container *corev1.Container)) func(podTemplateSpec *corev1.PodTemplateSpec) {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		if index >= len(podTemplateSpec.Spec.Containers) {
			podTemplateSpec.Spec.Containers = append(podTemplateSpec.Spec.Containers, corev1.Container{})
		}
		c := &podTemplateSpec.Spec.Containers[index]
		for _, f := range funcs {
			f(c)
		}
	}
}

// WithInitContainer applies the modifications to the init container with the provided name
func WithInitContainer(name string, containerfunc func(*corev1.Container)) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		idx := findIndexByName(name, podTemplateSpec.Spec.InitContainers)
		if idx == notFound {
			// if we are attempting to modify a container that does not exist, we will add a new one
			podTemplateSpec.Spec.InitContainers = append(podTemplateSpec.Spec.InitContainers, corev1.Container{Name: name})
			idx = len(podTemplateSpec.Spec.InitContainers) - 1
		}
		c := &podTemplateSpec.Spec.InitContainers[idx]
		containerfunc(c)
	}
}

// WithInitContainerByIndex applies the modifications to the container with the provided index
// if the index is out of range, a new container is added to accept these changes.
func WithInitContainerByIndex(index int, funcs ...func(container *corev1.Container)) func(podTemplateSpec *corev1.PodTemplateSpec) {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		if index >= len(podTemplateSpec.Spec.InitContainers) {
			podTemplateSpec.Spec.InitContainers = append(podTemplateSpec.Spec.InitContainers, corev1.Container{})
		}
		c := &podTemplateSpec.Spec.InitContainers[index]
		for _, f := range funcs {
			f(c)
		}
	}
}

// WithPodLabels sets the PodTemplateSpec's Labels
func WithPodLabels(labels map[string]string) Modification {
	if labels == nil {
		labels = map[string]string{}
	}
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.ObjectMeta.Labels = labels
	}
}

// WithServiceAccount sets the PodTemplateSpec's ServiceAccount name
func WithServiceAccount(serviceAccountName string) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.ServiceAccountName = serviceAccountName
	}
}

// WithVolume ensures the given volume exists
func WithVolume(volume corev1.Volume) Modification {
	return func(template *corev1.PodTemplateSpec) {
		for _, v := range template.Spec.Volumes {
			if v.Name == volume.Name {
				return
			}
		}
		template.Spec.Volumes = append(template.Spec.Volumes, volume)
	}
}

func findIndexByName(name string, containers []corev1.Container) int {
	for idx, c := range containers {
		if c.Name == name {
			return idx
		}
	}
	return notFound
}

// WithTerminationGracePeriodSeconds sets the PodTemplateSpec's termination grace period seconds
func WithTerminationGracePeriodSeconds(seconds int) Modification {
	s := int64(seconds)
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.TerminationGracePeriodSeconds = &s
	}
}

// WithSecurityContext sets the PodTemplateSpec's SecurityContext
func WithSecurityContext(securityContext corev1.PodSecurityContext) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		spec := &podTemplateSpec.Spec
		spec.SecurityContext = &securityContext
	}
}

// DefaultPodSecurityContext returns the default pod security context with FsGroup = 2000
func DefaultPodSecurityContext() corev1.PodSecurityContext {
	fsGroup := int64(2000)
	return corev1.PodSecurityContext{FSGroup: &fsGroup}
}

// WithImagePullSecrets adds an ImagePullSecrets local reference with the given name
func WithImagePullSecrets(name string) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		for _, v := range podTemplateSpec.Spec.ImagePullSecrets {
			if v.Name == name {
				return
			}
		}
		podTemplateSpec.Spec.ImagePullSecrets = append(podTemplateSpec.Spec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: name,
		})
	}
}

// WithTopologyKey sets the PodTemplateSpec's topology at a given index
func WithTopologyKey(topologyKey string, idx int) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[idx].PodAffinityTerm.TopologyKey = topologyKey
	}
}

// WithAffinity updates the name, antiAffinityLabelKey and weight of the PodTemplateSpec's Affinity
func WithAffinity(stsName, antiAffinityLabelKey string, weight int) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.Affinity =
			&corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
						Weight: int32(weight),
						PodAffinityTerm: corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{antiAffinityLabelKey: stsName}},
						},
					}},
				},
			}
	}
}

// WithNodeAffinity sets the PodTemplateSpec's node affinity
func WithNodeAffinity(nodeAffinity *corev1.NodeAffinity) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.Affinity.NodeAffinity = nodeAffinity
	}
}

// WithPodAffinity sets the PodTemplateSpec's pod affinity
func WithPodAffinity(podAffinity *corev1.PodAffinity) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.Affinity.PodAffinity = podAffinity
	}
}

// WithTolerations sets the PodTemplateSpec's tolerations
func WithTolerations(tolerations []corev1.Toleration) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Spec.Tolerations = tolerations
	}
}

// WithAnnotations sets the PodTemplateSpec's annotations
func WithAnnotations(annotations map[string]string) Modification {
	if annotations == nil {
		annotations = map[string]string{}
	}
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		podTemplateSpec.Annotations = annotations
	}
}

// WithVolumeMounts will add volume mounts to a container or init container by name
func WithVolumeMounts(containerName string, volumeMounts ...corev1.VolumeMount) Modification {
	return func(podTemplateSpec *corev1.PodTemplateSpec) {
		c := FindContainerByName(containerName, podTemplateSpec)
		if c == nil {
			return
		}
		container.WithVolumeMounts(volumeMounts)(c)
	}
}

// FindContainerByName will find either a container or init container by name in a pod template spec
func FindContainerByName(name string, podTemplateSpec *corev1.PodTemplateSpec) *corev1.Container {
	containerIdx := findIndexByName(name, podTemplateSpec.Spec.Containers)
	if containerIdx != notFound {
		return &podTemplateSpec.Spec.Containers[containerIdx]
	}

	initIdx := findIndexByName(name, podTemplateSpec.Spec.InitContainers)
	if initIdx != notFound {
		return &podTemplateSpec.Spec.InitContainers[initIdx]
	}

	return nil
}
