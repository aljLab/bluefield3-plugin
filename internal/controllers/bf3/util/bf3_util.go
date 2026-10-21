package util

import (
	"encoding/json"
	"github.com/mantra6g/bluefield3-plugin/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Bf3DriverConfig struct {
	ContainerName string
	DriverImage   string
	PodNamespace  string
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
		MatchLabels: map[string]string{v1alpha1.BlueField3TargetLabel: target.Name}, // TODO extract to method
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
	// spec.ServiceAccountName = ?
	// ensure that the driver is deployed on the Bluefield Host
	spec.Affinity = EnsureBf3DriverAffinity(*target.Spec.NodeName)
	spec.Containers = []corev1.Container{EnsureBf3DriverContainer(target, conf)}

	return spec
}

// choose node affinity to schedule the driver pod on the host where the BlueField is installed
func EnsureBf3DriverAffinity(nodeName string) *corev1.Affinity {
	affinity := &corev1.Affinity{}

	affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = GetBf3AffinityNodeSelectorTerms(nodeName)

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
	// might need more for moving interfaces from host ns -> pod net ns
	container.Args = []string{
		"--target-name", target.Name,
		"--max-nf-slots", "1",
	}
	// container.Env = EnsureDriverEnvVars(bmv2Target, container.Env) skip for now
	return container
}

func EnsureBf3PodMeta(target *v1alpha1.BlueField3Target, meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	if meta == nil {
		meta = &metav1.ObjectMeta{}
	}

	meta.Labels = map[string]string{v1alpha1.BlueField3TargetLabel: target.Name} // TODO extract to method
	meta.Annotations = GetBf3PodTemplateAnnotations(target)

	return meta
}

func GetBf3PodTemplateAnnotations(target *v1alpha1.BlueField3Target) map[string]string {
	return map[string]string{
		"k8s.v1.cni.cncf.io/networks": "[" + NewCNIConfigForTarget(target).String() + "]",
	}
}

type CNIConfig struct {
	Name      string  `json:"name"`
	Namespace string  `json:"namespace"`
	CNIArgs   CNIArgs `json:"cni-args"`
}

type CNIArgs struct {
	AppType      string `json:"app_type"`
	TargetName   string `json:"target_name"`
	NFInterfaces uint8  `json:"nf_interfaces"`
	MoveInterfaces []string `json:"move_interfaces"` // specify interfaces to be moved from the host net ns to the pod net ns
}

func (c CNIConfig) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func NewCNIConfigForTarget(target *v1alpha1.BlueField3Target) CNIConfig {
	return CNIConfig{
		Name:      "loom-cni",
		Namespace: "loom-system",
		CNIArgs: CNIArgs{
			AppType:      "HardwareDriver", // get a NetConfig Object back to configure the network
			TargetName:   target.Name,
			MoveInterfaces: []string{
				*target.Spec.InInterface,
				*target.Spec.OutInterface,
			},
		},
	}
}
