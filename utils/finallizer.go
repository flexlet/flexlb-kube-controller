package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const finalizer = "flexlb.gitee.io/finalizer"

func SetFinalizer(meta *metav1.ObjectMeta) bool {
	if !ListContains(meta.Finalizers, finalizer) {
		meta.Finalizers = append(meta.Finalizers, finalizer)
		return true
	}
	return false
}

func UnsetFinalizer(meta *metav1.ObjectMeta) bool {
	if ListContains(meta.Finalizers, finalizer) {
		meta.Finalizers = ListDelete(meta.Finalizers, finalizer)
		return true
	}
	return false
}
