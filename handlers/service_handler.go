package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	models "github.com/flexlet/flexlb-client-go/models"
	crdv1 "github.com/flexlet/flexlb-kube-controller/api/v1"
	"github.com/flexlet/flexlb-kube-controller/utils"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	utl "github.com/flexlet/utils"
)

// allocate flexlbinstance for load balancer type of service
func (h *Handler) ServiceChanged(k8s client.Client, ctx context.Context, svc *v1.Service) error {
	// old one is loadbalancer, but new one is not
	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		// delete instance recorded in annotation
		return h.deleteInstanceForService(k8s, ctx, svc)
	}

	// get cluster name annotation, set to default if not exist
	clusterName, exist := svc.Annotations[ClusterKey]
	if !exist {
		clusterName = DefaultClusterName
	}

	// get ippool annotation, set to default if not exist
	ippoolName, exist := svc.Annotations[IPPoolKey]
	if !exist {
		ippoolName = DefaultIPPoolName
	}

	ippool, err := getIPPool(k8s, ctx, h.namespace, clusterName, ippoolName)
	if err != nil {
		return h.errorf(svc, ErrorNoIPPool, err, "ippool does not exist")
	}

	// get service nodeIp endpoints
	endpoints, err := getNodeIpEndpoints(k8s, ctx, svc, ippool.BackendNetwork)
	if err != nil || len(endpoints) == 0 {
		return fmt.Errorf("not found node ip endpoint")
	}

	// get instance annotation
	if instName, exist := svc.Annotations[InstanceKey]; exist {
		// already allocated an instance, need to fetch and update
		inst := &crdv1.FlexLBInstance{}
		if err := k8s.Get(ctx, types.NamespacedName{Name: instName, Namespace: svc.Namespace}, inst); err != nil {
			// instance allocated but not exist in system, create a new one
			return h.createIntanceForService(k8s, ctx, svc, clusterName, ippoolName, endpoints)
		}
		// got the instance, check whether need update
		if needUpdate(inst, clusterName, ippoolName, endpoints) {
			// update instance and service
			return h.updateIntanceForService(k8s, ctx, svc, inst, clusterName, ippoolName, endpoints)
		}
		// no need of update
		return nil
	}

	// create instance and update service
	return h.createIntanceForService(k8s, ctx, svc, clusterName, ippoolName, endpoints)
}

func (h *Handler) ServiceDeleted(k8s client.Client, ctx context.Context, svc *v1.Service) error {
	return h.deleteInstanceForService(k8s, ctx, svc)
}

// find flexlbinstance from service annotation and delete it
// restriction: annotation should not be cleared manually
// another way is list instance with service annnotation, but has same restriction
func (h *Handler) deleteInstanceForService(k8s client.Client, ctx context.Context, svc *v1.Service) error {
	h.lock("delete balancer for service", "handler", "ServiceDeleted", "service", svc.Name, "namespace", svc.Namespace)
	defer h.unlock("delete balancer for service end", "handler", "ServiceDeleted", "service", svc.Name, "namespace", svc.Namespace)

	// get instance annotation
	if instName, exist := svc.Annotations[InstanceKey]; exist {
		// get and delete instance
		inst := &crdv1.FlexLBInstance{}
		if err := k8s.Get(ctx, types.NamespacedName{Name: instName, Namespace: svc.Namespace}, inst); err == nil {
			// instance exist, delete it
			if err1 := k8s.Delete(ctx, inst); err1 != nil {
				return err1
			}
		}
		// delete service annotation key
		delete(svc.Annotations, InstanceKey)
		return k8s.Update(ctx, svc)
	}
	return nil
}

// check whether instance need update
func needUpdate(inst *crdv1.FlexLBInstance, clusterName string, ippoolName string, endpoints []*models.Endpoint) bool {
	return (inst.Spec.Cluster != clusterName ||
		inst.Spec.IPPool != ippoolName ||
		!cmp.Equal(inst.Spec.Config.Endpoints, endpoints))
}

// get pod residents node's ip:port endpoints
func getNodeIpEndpoints(k8s client.Client, ctx context.Context, svc *v1.Service, trafficNetwork string) ([]*models.Endpoint, error) {
	flexlbEndpoints := []*models.Endpoint{}
	eps, err := utils.GetEndpointSliceOfService(k8s, ctx, svc)
	if err != nil {
		return flexlbEndpoints, err
	}
	backendDefaultOptions := "inter 2s downinter 5s rise 2 fall 2 slowstart 60s maxconn 2000 maxqueue 2000 weight 100 check"
	for _, port := range svc.Spec.Ports {
		backends := []*models.BackendServer{}
		for _, ep := range eps.Endpoints {
			trafficNodeIp, err := getNodeTrafficIp(k8s, ctx, *ep.NodeName, trafficNetwork)
			if err != nil {
				// by pass node with no traffic node ip
				continue
			}
			backend := &models.BackendServer{
				Name:      ep.TargetRef.Name,
				Ipaddress: *trafficNodeIp,
				Port:      uint16(port.NodePort),
			}
			backends = append(backends, backend)
		}

		// get protocol
		var mode string
		if port.Protocol == v1.ProtocolUDP {
			mode = models.EndpointModeUDP
		} else if port.Protocol == v1.ProtocolTCP {
			mode = models.EndpointModeTCP
		} else {
			// v1.ProtocolSCTP not support
			log.Log.Info("Service '%s' protocol not support", svc.Name)
			continue
		}

		flexlbEndpoint := &models.Endpoint{
			FrontendPort:         uint16(port.Port),
			Mode:                 mode,
			Balance:              "roundrobin",
			BackendOptions:       []string{},
			BackendDefaultServer: &backendDefaultOptions,
			BackendServers:       backends,
		}
		flexlbEndpoints = append(flexlbEndpoints, flexlbEndpoint)
	}
	return flexlbEndpoints, nil
}

// get node traffic ip from the node network annotation
func getNodeTrafficIp(k8s client.Client, ctx context.Context, nodeName string, trafficNetwork string) (*string, error) {
	node := &v1.Node{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: nodeName}, node); err != nil {
		return nil, err
	}
	data, exist := node.Annotations[NodeNetworkKey]
	if !exist {
		return nil, fmt.Errorf("node '%s' has no traffic network", nodeName)
	}

	nodeNets := []NodeNetwork{}

	if err := json.Unmarshal([]byte(data), &nodeNets); err != nil {
		return nil, fmt.Errorf("node '%s' has no traffic network", nodeName)
	}

	for _, nodeNet := range nodeNets {
		if nodeNet.Network == trafficNetwork {
			return &nodeNet.IPAddress, nil
		}
	}

	return nil, fmt.Errorf("node '%s' has no traffic network", nodeName)
}

// create instance and update service
func (h *Handler) createIntanceForService(k8s client.Client, ctx context.Context, svc *v1.Service,
	clusterName string, ippoolName string, endpoints []*models.Endpoint) error {

	h.lock("set balancer for service", "handler", "ServiceChanged", "service", svc.Name, "namespace", svc.Namespace)
	defer h.unlock("set balancer for service end", "handler", "ServiceChanged", "service", svc.Name, "namespace", svc.Namespace)

	// create instance
	inst, err := createIntance(k8s, ctx, h.namespace, clusterName, ippoolName, svc.Name, svc.Namespace, endpoints)
	if err != nil {
		return err
	}

	// update service annotaion
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[InstanceKey] = inst.Name
	if err := k8s.Update(ctx, svc); err != nil {
		return err
	}

	// update service loadbalancer
	ingress := v1.LoadBalancerIngress{IP: inst.Spec.Config.FrontendIpaddress}
	svc.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{ingress}
	return k8s.Status().Update(ctx, svc)
}

// update instance and update service
func (h *Handler) updateIntanceForService(k8s client.Client, ctx context.Context, svc *v1.Service, inst *crdv1.FlexLBInstance,
	clusterName string, ippoolName string, endpoints []*models.Endpoint) error {

	h.lock("update balancer for service", "handler", "ServiceChanged", "service", svc.Name, "namespace", svc.Namespace)
	defer h.unlock("update balancer for service end", "handler", "ServiceChanged", "service", svc.Name, "namespace", svc.Namespace)

	// update instance
	inst, err := updateIntance(k8s, ctx, inst, h.namespace, clusterName, ippoolName, endpoints)
	if err != nil {
		return err
	}

	// update service loadbalancer
	ingress := v1.LoadBalancerIngress{IP: inst.Spec.Config.FrontendIpaddress}
	svc.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{ingress}
	return k8s.Status().Update(ctx, svc)
}

// create flexlbinstance for service
func createIntance(k8s client.Client, ctx context.Context, flexlbNamespace string, clusterName string, ippoolName string,
	serviceName string, serviceNamespace string, endpoints []*models.Endpoint) (*crdv1.FlexLBInstance, error) {
	cluster := &crdv1.FlexLBCluster{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: flexlbNamespace}, cluster); err != nil {
		return nil, fmt.Errorf("cluster '%s' does not exist", clusterName)
	}
	var ippool *crdv1.FlexLBIPPool
	for i := 0; i < len(cluster.Spec.IPPools); i++ {
		if cluster.Spec.IPPools[i].Name == ippoolName {
			ippool = &cluster.Spec.IPPools[i]
			break
		}
	}
	if ippool == nil {
		return nil, fmt.Errorf("ippool '%s' in cluster '%s' does not exist", ippoolName, clusterName)
	}

	allocated, err := getAllocatedIp(k8s, ctx, clusterName, ippoolName)
	if err != nil {
		return nil, err
	}

	frontendIpaddress, err := utl.AllocIPFromRange(ippool.Start, ippool.End, allocated)
	if err != nil {
		return nil, err
	}

	instName := fmt.Sprintf("%s-%s", serviceName, utl.RandomString(4))
	inst := &crdv1.FlexLBInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instName,
			Namespace:   serviceNamespace,
			Annotations: map[string]string{ServiceKey: serviceName},
		},
		Spec: crdv1.FlexLBInstanceSpec{
			Cluster: clusterName,
			IPPool:  ippoolName,
			Config: models.InstanceConfig{
				Name:              instName,
				FrontendInterface: ippool.Interface,
				FrontendNetPrefix: ippool.NetPrefix,
				FrontendIpaddress: *frontendIpaddress,
				Endpoints:         endpoints,
			},
		},
	}
	return inst, k8s.Create(ctx, inst)
}

// get ippool object by cluster name and ippool name
func getIPPool(k8s client.Client, ctx context.Context, flexlbNamespace string, clusterName string, ippoolName string) (*crdv1.FlexLBIPPool, error) {
	cluster := &crdv1.FlexLBCluster{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: flexlbNamespace}, cluster); err != nil {
		return nil, fmt.Errorf("cluster '%s' does not exist", clusterName)
	}
	var ippool *crdv1.FlexLBIPPool
	for i := 0; i < len(cluster.Spec.IPPools); i++ {
		if cluster.Spec.IPPools[i].Name == ippoolName {
			ippool = &cluster.Spec.IPPools[i]
			break
		}
	}
	if ippool == nil {
		return nil, fmt.Errorf("ippool '%s' in cluster '%s' does not exist", ippoolName, clusterName)
	}
	return ippool, nil
}

// update flexlbinstance for service
func updateIntance(k8s client.Client, ctx context.Context, inst *crdv1.FlexLBInstance, flexlbNamespace string,
	clusterName string, ippoolName string, endpoints []*models.Endpoint) (*crdv1.FlexLBInstance, error) {
	if inst.Spec.Cluster != clusterName || inst.Spec.IPPool != ippoolName {
		// cluster or ip pool changed, need to allocate new ip
		ippool, err := getIPPool(k8s, ctx, flexlbNamespace, clusterName, ippoolName)
		if err != nil {
			return nil, fmt.Errorf("ippool '%s' in cluster '%s' does not exist", ippoolName, clusterName)
		}

		allocated, err := getAllocatedIp(k8s, ctx, clusterName, ippoolName)
		if err != nil {
			return nil, err
		}

		frontendIpaddress, err := utl.AllocIPFromRange(ippool.Start, ippool.End, allocated)
		if err != nil {
			return nil, err
		}
		inst.Spec.Cluster = clusterName
		inst.Spec.IPPool = ippoolName
		inst.Spec.Config.FrontendInterface = ippool.Interface
		inst.Spec.Config.FrontendNetPrefix = ippool.NetPrefix
		inst.Spec.Config.FrontendIpaddress = *frontendIpaddress
	}

	inst.Spec.Config.Endpoints = endpoints
	return inst, k8s.Update(ctx, inst)
}

// list instance, find allocated ip
func getAllocatedIp(k8s client.Client, ctx context.Context, clusterName string, ippoolName string) ([]string, error) {
	allocated := []string{}

	insts := crdv1.FlexLBInstanceList{}
	instLabels := map[string]string{
		ClusterKey: clusterName,
		IPPoolKey:  ippoolName,
	}

	if err := k8s.List(ctx, &insts, client.MatchingLabels(instLabels)); err != nil {
		return allocated, fmt.Errorf("list exist instance failed: %s", err.Error())
	}

	for i := 0; i < len(insts.Items); i++ {
		allocated = append(allocated, insts.Items[i].Spec.Config.FrontendIpaddress)
	}
	return allocated, nil
}
