package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utl "github.com/flexlet/utils"
)

const finalizer = "flexlb.flexlet.io/finalizer"

func SetFinalizer(meta *metav1.ObjectMeta) bool {
	if !utl.ListContains(meta.Finalizers, finalizer) {
		meta.Finalizers = append(meta.Finalizers, finalizer)
		return true
	}
	return false
}

func UnsetFinalizer(meta *metav1.ObjectMeta) bool {
	if utl.ListContains(meta.Finalizers, finalizer) {
		meta.Finalizers = utl.ListDelete(meta.Finalizers, finalizer)
		return true
	}
	return false
}
