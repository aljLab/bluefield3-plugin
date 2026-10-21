package p4target

import (
	apiv1alpha1 "github.com/mantra6g/bluefield3-plugin/api/v1alpha1"
	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"
)

func EnsureP4TargetLabels(bf3Target *apiv1alpha1.BlueField3Target,
	labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[apiv1alpha1.BlueField3TargetLabel] = bf3Target.Name
	return labels
}

func EnsureP4TargetAnnotations(bf3Target *apiv1alpha1.BlueField3Target,
	annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	return annotations
}

func EnsureP4TargetFinalizers(bf3Target *apiv1alpha1.BlueField3Target,
	finalizers []string) []string {
	return finalizers
}

func EnsureP4TargetSpec(bf3Target *apiv1alpha1.BlueField3Target,
	spec *corev1alpha1.P4TargetSpec) *corev1alpha1.P4TargetSpec {
	if spec == nil {
		spec = &corev1alpha1.P4TargetSpec{}
	}
	return spec
}
