/*
Copyright 2022 FlexLB Project.
*/

package controllers

import (
	"context"

	"gitee.com/flexlb/flexlb-kube-controller/handlers"
	"gitee.com/flexlb/flexlb-kube-controller/utils"
	v1 "k8s.io/api/core/v1"
	disv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// FlexLBClusterReconciler reconciles a FlexLBCluster object
type ServiceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ChangeHandler func(client.Client, context.Context, *v1.Service) error
	DeleteHandler func(client.Client, context.Context, *v1.Service) error
}

//+kubebuilder:rbac:groups="",resources=service,verbs=get;list;watch;update;patch;
//+kubebuilder:rbac:groups="",resources=service/status,verbs=update;
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;
//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch;
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var service v1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		return ctrl.Result{}, nil
	}

	if service.ObjectMeta.DeletionTimestamp.IsZero() {
		// set finalizer if not set
		if utils.SetFinalizer(&service.ObjectMeta) {
			log.Log.Info("set finalizer ", "controller", "ServiceReconciler", "object", req.NamespacedName)
			if err := r.Update(ctx, &service); err != nil {
				log.Log.Info("set finalizer failed", "controller", "ServiceReconciler", "object", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}

		// process change request
		if err := r.ChangeHandler(r.Client, ctx, &service); err != nil {
			return ctrl.Result{}, err
		}

	} else {
		// process delete request
		if err := r.DeleteHandler(r.Client, ctx, &service); err != nil {
			return ctrl.Result{}, err
		}

		// unset finalizer if set
		if utils.UnsetFinalizer(&service.ObjectMeta) {
			log.Log.Info("unset finalizer ", "controller", "ServiceReconciler", "object", req.NamespacedName)
			if err := r.Update(ctx, &service); err != nil {
				log.Log.Info("unset finalizer failed", "controller", "ServiceReconciler", "object", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			svc := e.Object.(*v1.Service)
			return needReconcile(svc, r.Client)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*v1.Service)
			new := e.ObjectNew.(*v1.Service)
			// if old one is balancer but new one is not, need to delete instance
			return needReconcile(old, r.Client) || needReconcile(new, r.Client)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			svc := e.Object.(*v1.Service)
			return needReconcile(svc, r.Client)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			svc := e.Object.(*v1.Service)
			return needReconcile(svc, r.Client)
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Service{}, builder.WithPredicates(p)).
		Watches(&source.Kind{Type: &disv1.EndpointSlice{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				epSlice, ok := obj.(*disv1.EndpointSlice)
				if !ok {
					return []reconcile.Request{}
				}
				service, err := utils.GetServiceOfEndpointSlice(r.Client, context.TODO(), epSlice)
				if err != nil || service.Spec.Type != v1.ServiceTypeLoadBalancer {
					return []reconcile.Request{}
				}
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: service.Namespace, Name: service.Name}}}
			})).
		Complete(r)
}

// check whether service need reconcile
func needReconcile(svc *v1.Service, k8s client.Client) bool {
	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		// not balancer type
		return false
	}

	if _, err := utils.GetEndpointSliceOfService(k8s, context.TODO(), svc); err != nil {
		// no endpoint slice
		return false
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		// no allocated balancer
		return true
	}

	// already has allocated balancer, check if it's managed by flexlb (has instance annotation)
	_, byFlexlb := svc.Annotations[handlers.InstanceKey]

	// if it's managed by flexlb, go to reconciler
	return byFlexlb
}
