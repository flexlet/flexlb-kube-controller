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
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crdv1 "gitee.com/flexlb/flexlb-kube-controller/api/v1"
)

func (h *Handler) InstanceChanged(k8s client.Client, ctx context.Context, instance *crdv1.FlexLBInstance) error {
	h.lock("update instance", "handler", "InstanceChanged", "instance", instance.Name, "namespace", instance.Namespace)
	defer h.unlock("update instance end", "handler", "InstanceChanged", "instance", instance.Name, "namespace", instance.Namespace)

	// get service annotation
	if svcName, exist := instance.Annotations[ServiceKey]; exist {
		// check service exist or not
		svc := &v1.Service{}
		if err := k8s.Get(ctx, types.NamespacedName{Name: svcName, Namespace: instance.Namespace}, svc); err != nil {
			// service not exist, delete instance
			h.errorf(instance, ErrorInvalidConfig, nil, "instance deleted because invalid config: service not exist")
			return k8s.Delete(ctx, instance)
		}
	}

	// get owned cluster
	cluster, err1 := getOwnedCluster(k8s, ctx, instance, h.namespace)
	if err1 != nil {
		// delete instance if cluster not exist
		h.errorf(instance, ErrorInvalidConfig, nil, "instance deleted because invalid config: cluster not exist")
		return k8s.Delete(ctx, instance)
	}

	// get owned IP pool
	ippool, err2 := getOwnedIPPool(instance, cluster)
	if err2 != nil {
		// delete instance if ippool not exist
		h.errorf(instance, ErrorInvalidConfig, nil, "instance deleted because invalid config: ippool not exist")
		return k8s.Delete(ctx, instance)
	}

	// check if instance config matches ippool
	if !ippool.Matches(&instance.Spec.Config) {
		// delete instance if ip not in ippool
		h.errorf(instance, ErrorInvalidConfig, nil, "instance deleted because invalid config: frontend does not match ippool")
		return k8s.Delete(ctx, instance)
	}

	// connect cluster and update cluster status
	lb, err3 := h.connectCluster(k8s, ctx, cluster)
	if err3 != nil {
		// connect failed, update intance status
		updateInstanceStatus(k8s, ctx, instance, crdv1.InstancePhaseClusterNotReady, nil)
		return h.errorf(instance, ErrorClusterNotReady, err3, "cluster not ready")
	}

	// check if exist
	exist, _ := lb.GetInstance(instance.Spec.Config.Name)
	if exist != nil {
		// config same, load exist status
		if cmp.Equal(exist.Config, &instance.Spec.Config) {
			phase := crdv1.InstancePhaseNotReady
			for _, v := range exist.Status {
				if v == crdv1.InstanceStatusUp {
					phase = crdv1.InstancePhaseReady
					break
				}
			}
			// update instance status
			updateInstanceStatus(k8s, ctx, instance, phase, &exist.Status)

			// if status not ready, retry later
			if phase != crdv1.InstancePhaseReady {
				return h.errorf(instance, ErrorInstanceNotReady, nil, "instance not ready")
			}

			// status ready
			return nil
		}

		// config not same, modify exist
		if modified, err := lb.ModifyInstance(&instance.Spec.Config); err != nil {
			// modify failed, update instance status
			updateInstanceStatus(k8s, ctx, instance, crdv1.InstancePhaseModifyFailed, nil)
			// retry later
			return h.errorf(instance, ErrorInstanceModifyFailed, err, "instance modify failed")
		} else {
			// modify succeed, update instance labels & status
			updateInstanceLabels(k8s, ctx, instance)
			updateInstanceStatus(k8s, ctx, instance, crdv1.InstancePhaseModified, &modified.Status)
			return nil
		}
	}

	// not exist, create new one
	created, err4 := lb.CreateInstance(&instance.Spec.Config)
	if err4 != nil {
		// create failed, update instance status
		updateInstanceStatus(k8s, ctx, instance, crdv1.InstancePhaseCreateFailed, nil)
		return h.errorf(instance, ErrorInstanceCreateFailed, err4, "instance create failed")
	}

	// create succeed, update instance labels & status
	updateInstanceLabels(k8s, ctx, instance)
	updateInstanceStatus(k8s, ctx, instance, crdv1.InstancePhaseCreated, &created.Status)

	return nil
}

func (h *Handler) InstanceDeleted(k8s client.Client, ctx context.Context, instance *crdv1.FlexLBInstance) error {
	h.lock("delete instance", "handler", "InstanceDeleted", "instance", instance.Name, "namespace", instance.Namespace)
	defer h.unlock("delete instance end", "handler", "InstanceDeleted", "instance", instance.Name, "namespace", instance.Namespace)

	// get owned cluster
	cluster, err1 := getOwnedCluster(k8s, ctx, instance, h.namespace)
	if err1 != nil {
		// cluster not exist, delete directly
		return nil
	}

	// connect cluster and update cluster status
	lb, err3 := h.connectCluster(k8s, ctx, cluster)
	if err3 != nil {
		// connect failed, delete directly
		return nil
	}

	// check if exist
	exist, _ := lb.GetInstance(instance.Spec.Config.Name)
	if exist != nil {
		// delete if exist
		lb.DeleteInstance(exist.Config.Name)
	}

	// not exist, or delete failed, delete directly
	return nil
}

func updateInstanceLabels(k8s client.Client, ctx context.Context, instance *crdv1.FlexLBInstance) error {
	if instance.Labels == nil {
		instance.Labels = map[string]string{}
	}

	if instance.Spec.Cluster != "" {
		instance.Labels[ClusterKey] = instance.Spec.Cluster
	} else {
		instance.Labels[ClusterKey] = DefaultClusterName
	}

	if instance.Spec.IPPool != "" {
		instance.Labels[IPPoolKey] = instance.Spec.IPPool
	} else {
		instance.Labels[IPPoolKey] = DefaultIPPoolName
	}

	return k8s.Update(ctx, instance)
}

func updateInstanceStatus(k8s client.Client, ctx context.Context, instance *crdv1.FlexLBInstance, phase string, nodeStatus *map[string]string) error {
	if nodeStatus != nil {
		instance.Status = crdv1.FlexLBInstanceStatus{Phase: phase, NodeStatus: *nodeStatus}
	} else {
		instance.Status = crdv1.FlexLBInstanceStatus{Phase: phase}
	}
	return k8s.Status().Update(ctx, instance)
}

// get the owned cluster of instance
func getOwnedCluster(k8s client.Client, ctx context.Context, instance *crdv1.FlexLBInstance, namespace string) (*crdv1.FlexLBCluster, error) {
	var clusterNamespacedName types.NamespacedName
	if instance.Spec.Cluster != "" {
		clusterNamespacedName = types.NamespacedName{Namespace: namespace, Name: instance.Spec.Cluster}
	} else {
		clusterNamespacedName = types.NamespacedName{Namespace: namespace, Name: DefaultClusterName}
	}

	var cluster crdv1.FlexLBCluster
	if err := k8s.Get(ctx, clusterNamespacedName, &cluster); err != nil {
		return nil, err
	}
	return &cluster, nil
}

// get the owned ippool of instance
func getOwnedIPPool(instance *crdv1.FlexLBInstance, cluster *crdv1.FlexLBCluster) (*crdv1.FlexLBIPPool, error) {
	var IPPoolName = DefaultIPPoolName
	if instance.Spec.IPPool != "" {
		IPPoolName = instance.Spec.IPPool
	}

	for i := 0; i < len(cluster.Spec.IPPools); i++ {
		if IPPoolName == cluster.Spec.IPPools[i].Name {
			return &cluster.Spec.IPPools[i], nil
		}
	}
	return nil, fmt.Errorf("IP pool '%s' does not exist on owned cluster '%s' of instance '%s'", IPPoolName, instance.Name, cluster.Name)
}
