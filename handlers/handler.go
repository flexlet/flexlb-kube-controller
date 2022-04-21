// Copyright (c) 2022 Yaohui Wang (yaohuiwang@outlook.com)
// FlexLB is licensed under Mulan PubL v2.
// You can use this software according to the terms and conditions of the Mulan PubL v2.
// You may obtain a copy of Mulan PubL v2 at:
//         http://license.coscl.org.cn/MulanPubL-2.0
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
// EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
// MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
// See the Mulan PubL v2 for more details.

package handlers

import (
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/deprecated/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Handler struct {
	tlsCaCert     string
	tlsClientCert string
	tlsClientKey  string
	tlsInsecure   bool
	namespace     string
	recorder      record.EventRecorder
	sync.Mutex
}

func NewHandler(tlsCaCert string, tlsClientCert string, tlsClientKey string, tlsInsecure bool, namespace string, recorder record.EventRecorder) *Handler {
	return &Handler{
		tlsCaCert:     tlsCaCert,
		tlsClientCert: tlsClientCert,
		tlsClientKey:  tlsClientKey,
		tlsInsecure:   tlsInsecure,
		namespace:     namespace,
		recorder:      recorder,
	}
}

const (
	DefaultClusterName = "default"
	DefaultIPPoolName  = "default"
)

// instance errors
const (
	ErrorInvalidConfig        = "ErrorInvalidConfig"
	ErrorClusterNotReady      = "ErrorClusterNotReady"
	ErrorInstanceNotReady     = "ErrorInstanceNotReady"
	ErrorInstanceModifyFailed = "ErrorInstanceModifyFailed"
	ErrorInstanceCreateFailed = "ErrorInstanceCreateFailed"
	ErrorInstanceDeleteFailed = "ErrorInstanceDeleteFailed"
)

// instance annotation keys
const (
	ServiceKey = "flexlb.gitee.io/service"
)

// node errors
const (
	ErrorProbeTrafficNodeIp  = "ErrorProbeTrafficNodeIp"
	ErrorUpdateTrafficNodeIp = "ErrorUpdateTrafficNodeIp"
)

// node annotation key
const (
	NodeNetworkKey = "flexlb.gitee.io/nodeNetwork"
)

// node network annotation struct
type NodeNetwork struct {
	Network   string `json:"network,omitempty"`
	Device    string `json:"device,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// service annotation keys
const (
	ClusterKey  = "flexlb.gitee.io/cluster"
	IPPoolKey   = "flexlb.gitee.io/ippool"
	InstanceKey = "flexlb.gitee.io/instance"
)

const (
	ErrorNoIPPool = "ErrorNoIPPool"
)

func (h *Handler) lock(msg string, kvs ...interface{}) {
	log.Log.Info(msg, kvs...)
	h.Lock()
}

func (h *Handler) unlock(msg string, kvs ...interface{}) {
	h.Unlock()
	log.Log.Info(msg, kvs...)
}

func (h *Handler) errorf(object runtime.Object, reason string, err error, msgfmt string, args ...interface{}) error {
	ref, _ := reference.GetReference(scheme.Scheme, object)

	msg := fmt.Sprintf(msgfmt, args...)
	if err != nil {
		msg = msg + ": " + err.Error()
	}

	h.recorder.Event(ref, v1.EventTypeWarning, reason, msg)

	return fmt.Errorf(msg)
}
