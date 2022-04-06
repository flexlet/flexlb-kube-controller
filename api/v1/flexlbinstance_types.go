/*
Copyright 2022 FlexLB Project.
*/

package v1

import (
	"bytes"
	"net"

	models "gitee.com/flexlb/flexlb-client-go/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FlexLBInstanceSpec defines the desired state of FlexLBInstance
type FlexLBInstanceSpec struct {
	Cluster string                `json:"cluster,omitempty"`
	IPPool  string                `json:"ippool,omitempty"`
	Config  models.InstanceConfig `json:"config,omitempty"`
}

// FlexLBInstanceStatus defines the observed state of FlexLBInstance
type FlexLBInstanceStatus struct {
	Phase      string            `json:"phase"`
	NodeStatus map[string]string `json:"node_status"`
}

const (
	InstancePhaseClusterNotReady = "cluster_not_ready"
	InstancePhaseCreated         = "created"
	InstancePhaseCreateFailed    = "create_failed"
	InstancePhaseModified        = "modified"
	InstancePhaseModifyFailed    = "modify_failed"
	InstancePhaseReady           = "ready"
	InstancePhaseNotReady        = "not_ready"
)

const (
	InstanceStatusUp      = "up"
	InstanceStatusDown    = "down"
	InstanceStatusPending = "pending"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FlexLBInstance is the Schema for the flexlbinstances API
type FlexLBInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlexLBInstanceSpec   `json:"spec,omitempty"`
	Status FlexLBInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FlexLBInstanceList contains a list of FlexLBInstance
type FlexLBInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlexLBInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FlexLBInstance{}, &FlexLBInstanceList{})
}

func (in *FlexLBInstanceSpec) DeepCopyInto(out *FlexLBInstanceSpec) {
	*out = *in
}

func (in *FlexLBInstanceSpec) DeepCopy() *FlexLBInstanceSpec {
	if in == nil {
		return nil
	}
	out := new(FlexLBInstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// ippool matches instance config
func (p *FlexLBIPPool) Matches(cfg *models.InstanceConfig) bool {
	if cfg.FrontendInterface != p.Interface {
		return false
	}
	if cfg.FrontendNetPrefix != p.NetPrefix {
		return false
	}

	// check instance frontend ipaddress in IP pool
	ip := net.ParseIP(cfg.FrontendIpaddress)
	ip1 := net.ParseIP(p.Start)
	ip2 := net.ParseIP(p.End)
	if bytes.Compare(ip, ip1) < 0 || bytes.Compare(ip, ip2) > 0 {
		return false
	}

	return true
}
