/*
Copyright 2022 FlexLB Project.
*/

package v1

import (
	models "gitee.com/flexlb/flexlb-client-go/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FlexLB Cluster IP Pools
type FlexLBIPPool struct {
	Name           string `json:"name,omitempty"`
	Interface      string `json:"interface,omitempty"`
	NetPrefix      uint8  `json:"net_prefix,omitempty"`
	Start          string `json:"start,omitempty"`
	End            string `json:"end,omitempty"`
	BackendNetwork string `json:"backend_network,omitempty"`
}

// FlexLBClusterSpec defines the desired state of FlexLBCluster
type FlexLBClusterSpec struct {
	IPPools  []FlexLBIPPool `json:"ippools,omitempty"`
	Endpoint string         `json:"endpoint,omitempty"`
}

// FlexLBClusterStatus defines the observed state of FlexLBCluster
type FlexLBClusterStatus struct {
	// cluster ready status
	ClusterStatus string `json:"cluster_status,omitempty"`

	// FlexLBNode ready status, example: {node1: ready, node2: ready}
	NodeStatus models.ReadyStatus `json:"node_status,omitempty"`
}

const (
	ClusterStatusReady    = "ready"
	ClusterStatusNotReady = "not_ready"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FlexLBCluster is the Schema for the flexlbclusters API
type FlexLBCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlexLBClusterSpec   `json:"spec,omitempty"`
	Status FlexLBClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FlexLBClusterList contains a list of FlexLBCluster
type FlexLBClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlexLBCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FlexLBCluster{}, &FlexLBClusterList{})
}
