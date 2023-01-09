package utils

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	disv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// get the endpoint slice of service
func GetEndpointSliceOfService(k8s client.Client, ctx context.Context, svc *v1.Service) (*disv1.EndpointSlice, error) {
	serviceLabel := map[string]string{
		disv1.LabelServiceName: svc.Name,
	}
	epslist := &disv1.EndpointSliceList{}
	k8s.List(ctx, epslist, client.InNamespace(svc.Namespace), client.MatchingLabels(serviceLabel))
	if len(epslist.Items) == 0 {
		return nil, fmt.Errorf("service '%s/%s' has no endpointslice", svc.Namespace, svc.Name)
	}
	return &epslist.Items[0], nil
}

// get the service name from endpointslice label
func GetServiceOfEndpointSlice(k8s client.Client, ctx context.Context, endpointSlice *disv1.EndpointSlice) (*v1.Service, error) {
	serviceName, exist := endpointSlice.Labels[disv1.LabelServiceName]
	if !exist || serviceName == "" {
		return nil, fmt.Errorf("endpointSlice missing the %s label", disv1.LabelServiceName)
	}
	service := &v1.Service{}
	serviceKey := types.NamespacedName{Namespace: endpointSlice.Namespace, Name: serviceName}
	if err := k8s.Get(ctx, serviceKey, service); err != nil {
		return nil, err
	}
	return service, nil
}
