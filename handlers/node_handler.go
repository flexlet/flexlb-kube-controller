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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitee.com/flexlb/flexlb-kube-controller/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// add node network annotation
func (h *Handler) NodeChanged(k8s client.Client, ctx context.Context, node *v1.Node) error {
	nodeNetwork, err := getNodeNetwork(k8s, ctx, node.Name, h.namespace)
	if err != nil {
		return h.errorf(node, ErrorProbeTrafficNodeIp, err, "probe node network failed")
	}
	node.Annotations[NodeNetworkKey] = *nodeNetwork
	if err := k8s.Update(ctx, node); err != nil {
		return h.errorf(node, ErrorUpdateTrafficNodeIp, err, "update node annotation failed")
	}
	return nil
}

// create a pod on target node to get host network
const (
	probePodNamePrefix = "flexlb-node-probe-"
	probePodImage      = "busybox"
	probePodCommand    = "ip route | awk \"/dev.*src/{print \\$1,\\$3,\\$(NF-2)}\""
	probePodTimeout    = 10
)

// delete pod if exist
func delPodIfExist(k8s client.Client, ctx context.Context, podKey types.NamespacedName) {
	pod := &v1.Pod{}
	if err := k8s.Get(ctx, podKey, pod); err == nil {
		k8s.Delete(ctx, pod)
	}
}

// get node network (json string of [{'network':<cidr>,'device':<dev>,'ip_address':<ip>}])
func getNodeNetwork(k8s client.Client, ctx context.Context, nodeName string, namespace string) (*string, error) {
	// create a pod on target node (use host network)
	automountServiceAccountToken := false
	probePod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      probePodNamePrefix + nodeName,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			NodeName:                     nodeName,
			HostNetwork:                  true,
			RestartPolicy:                v1.RestartPolicyOnFailure,
			AutomountServiceAccountToken: &automountServiceAccountToken,
			Containers: []v1.Container{
				{
					Name:            probePodNamePrefix + nodeName,
					Image:           probePodImage,
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         []string{"sh", "-c", probePodCommand},
				},
			},
		},
	}

	probePodKey := types.NamespacedName{
		Name:      probePodNamePrefix + nodeName,
		Namespace: namespace,
	}

	// delete probe pod if exist (last time failed in middle of reconcile)
	delPodIfExist(k8s, ctx, probePodKey)

	// create probe pod
	if err := k8s.Create(ctx, probePod); err != nil {
		return nil, err
	}
	defer k8s.Delete(ctx, probePod)

	// wait util success
	execSuccess := false
	for i := 0; i < probePodTimeout*100; i++ {
		time.Sleep(time.Second / 100)
		if err := k8s.Get(ctx, probePodKey, probePod); err != nil {
			// not created yet
			continue
		}
		if len(probePod.Status.ContainerStatuses) == 0 {
			// container not created yet
			continue
		}
		doneState := probePod.Status.ContainerStatuses[0].State.Terminated
		if doneState == nil {
			// not complate yet
			continue
		}

		if doneState.ExitCode != 0 {
			// exec failed
			return nil, fmt.Errorf("probe pod exit with code %d, terminate reason: %s", doneState.ExitCode, doneState.Reason)
		} else {
			execSuccess = true
			break
		}
	}

	// wait timeout
	if !execSuccess {
		return nil, fmt.Errorf("probe pod exec timeout")
	}

	// get logs
	podLogs, err := utils.GetPodLogs(ctx, probePod.Namespace, probePod.Name)
	if err != nil || *podLogs == "" {
		return nil, fmt.Errorf("get probe pod logs failed: %s", err.Error())
	}

	// parse logs
	nodeNetworks := []NodeNetwork{}
	lines := strings.Split(*podLogs, "\n")
	for _, line := range lines {
		entries := strings.Split(line, " ")
		if len(entries) != 3 {
			continue
		}
		nodeNetwork := NodeNetwork{
			Network:   entries[0],
			Device:    entries[1],
			IPAddress: entries[2],
		}
		nodeNetworks = append(nodeNetworks, nodeNetwork)
	}
	if len(nodeNetworks) == 0 {
		return nil, fmt.Errorf("get node network failed")
	}

	// array to json
	data, _ := json.Marshal(nodeNetworks)
	nets := string(data)
	return &nets, nil
}
