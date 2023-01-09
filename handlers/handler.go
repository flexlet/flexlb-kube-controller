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
	probePodImage string
	recorder      record.EventRecorder
	sync.Mutex
}

func NewHandler(tlsCaCert string, tlsClientCert string, tlsClientKey string, tlsInsecure bool, namespace string, probePodImage string, recorder record.EventRecorder) *Handler {
	return &Handler{
		tlsCaCert:     tlsCaCert,
		tlsClientCert: tlsClientCert,
		tlsClientKey:  tlsClientKey,
		tlsInsecure:   tlsInsecure,
		namespace:     namespace,
		probePodImage: probePodImage,
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
	ServiceKey = "flexlb.flexlet.io/service"
)

// node errors
const (
	ErrorProbeTrafficNodeIp  = "ErrorProbeTrafficNodeIp"
	ErrorUpdateTrafficNodeIp = "ErrorUpdateTrafficNodeIp"
)

// node annotation key
const (
	NodeNetworkKey = "flexlb.flexlet.io/nodeNetwork"
)

// node network annotation struct
type NodeNetwork struct {
	Network   string `json:"network,omitempty"`
	Device    string `json:"device,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// service annotation keys
const (
	ClusterKey  = "flexlb.flexlet.io/cluster"
	IPPoolKey   = "flexlb.flexlet.io/ippool"
	InstanceKey = "flexlb.flexlet.io/instance"
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
