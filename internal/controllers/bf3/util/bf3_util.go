package util

import (
	// "encoding/json"
	"github.com/mantra6g/bluefield3-plugin/api/v1alpha1"
	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Bf3DriverConfig struct {
	ContainerName        string
	DriverImage          string
	PodNamespace         string
	DriverServiceAccount string
}

/*
spec:

	network:
		srv6:
			mode: Proxy
		hostInterfaces:
			strategy: Reuse
			reuse:
				names: ["bf0", "bf1"] -> get from BF3TargetSpec
*/
func EnsureP4TargetSpec(target *v1alpha1.BlueField3Target) *corev1alpha1.P4TargetSpec {
	spec := &corev1alpha1.P4TargetSpec{}
	spec.Network = EnsureP4TargetNetwork(target)
	return spec
}

func EnsureP4TargetNetwork(target *v1alpha1.BlueField3Target) *corev1alpha1.P4TargetNetwork {
	p4net := &corev1alpha1.P4TargetNetwork{}
	p4net.SRv6 = &target.Spec.SRv6ProxyConfig
	p4net.HostInterfaces = &corev1alpha1.HostInterfaces{
		Strategy: "ReuseMapped",
		ReuseMapped: &corev1alpha1.ReuseMappedHostInterfaces{
			IfacePairs: target.Spec.Interfaces,
		},
	}
	return p4net
}

func EnsureBf3DeploymentLabels(target *v1alpha1.BlueField3Target, labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[v1alpha1.BlueField3TargetLabel] = target.Name
	return labels
}

func EnsureBf3DeploymentAnnotations(target *v1alpha1.BlueField3Target, annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	return annotations
}

func EnsureBf3DeploymentFinalizers(target *v1alpha1.BlueField3Target, finalizers []string) []string {
	return finalizers
}

func EnsureBf3DeploymentSpec(target *v1alpha1.BlueField3Target, spec *appsv1.DeploymentSpec, conf *Bf3DriverConfig) *appsv1.DeploymentSpec {
	if spec == nil {
		spec = &appsv1.DeploymentSpec{}
	}
	spec.Replicas = new(int32(1))
	spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{v1alpha1.BlueField3TargetLabel: target.Name},
	}
	spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: *EnsureBf3PodMeta(target, &spec.Template.ObjectMeta),
		Spec:       *EnsureBf3PodSpec(target, &spec.Template.Spec, conf),
	}
	return spec
}

func EnsureBf3PodSpec(target *v1alpha1.BlueField3Target, spec *corev1.PodSpec, conf *Bf3DriverConfig) *corev1.PodSpec {
	if spec == nil {
		spec = &corev1.PodSpec{}
	}
	spec.ServiceAccountName = conf.DriverServiceAccount

	// ensure that the driver is deployed on the Bluefield Host
	spec.Affinity = EnsureBf3DriverAffinity(*target.Spec.NodeName)
	spec.Containers = []corev1.Container{EnsureBf3DriverContainer(target, conf)}

	return spec
}

// choose node affinity to schedule the driver pod on the host where the BlueField is installed
func EnsureBf3DriverAffinity(nodeName string) *corev1.Affinity {
	affinity := &corev1.Affinity{}
	nodeAffinity := corev1.NodeAffinity{}
	req := corev1.NodeSelector{}
	nodeSelectorTerms := GetBf3AffinityNodeSelectorTerms(nodeName)

	req.NodeSelectorTerms = nodeSelectorTerms
	nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &req
	affinity.NodeAffinity = &nodeAffinity

	return affinity
}

func GetBf3AffinityNodeSelectorTerms(name string) []corev1.NodeSelectorTerm {
	return []corev1.NodeSelectorTerm{
		{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "kubernetes.io/hostname",
					Operator: "In",
					Values:   []string{name},
				},
			},
		},
	}
}

func EnsureBf3DriverContainer(target *v1alpha1.BlueField3Target, conf *Bf3DriverConfig) corev1.Container {
	container := corev1.Container{}
	container.Name = conf.ContainerName
	container.Image = conf.DriverImage

	container.ImagePullPolicy = corev1.PullIfNotPresent
	if container.SecurityContext == nil {
		container.SecurityContext = &corev1.SecurityContext{}
	}
	if container.SecurityContext.Capabilities == nil {
		container.SecurityContext.Capabilities = &corev1.Capabilities{}
	}
	container.SecurityContext.Capabilities.Add = []corev1.Capability{"NET_ADMIN"}

	container.Args = append([]string{
		"--target-name", target.Name,
		"--max-nf-slots", "1",
	}, target.Spec.DriverArgs...)
	return container
}

func EnsureBf3PodMeta(target *v1alpha1.BlueField3Target, meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	if meta == nil {
		meta = &metav1.ObjectMeta{}
	}

	meta.Labels = map[string]string{v1alpha1.BlueField3TargetLabel: target.Name}
	meta.Annotations = map[string]string{}

	return meta
}
