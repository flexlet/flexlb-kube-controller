// Copyright (c) 2022 Yaohui Wang (yaohuiwang@outlook.com)
// FlexLB is licensed under Mulan PubL v2.
// You can use this software according to the terms and conditions of the Mulan PubL v2.
// You may obtain a copy of Mulan PubL v2 at:
//         http://license.coscl.org.cn/MulanPubL-2.0
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
// EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
// MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
// See the Mulan PubL v2 for more details.

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utl "github.com/00ahui/utils"
)

const finalizer = "flexlb.gitee.io/finalizer"

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
