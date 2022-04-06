/*
Copyright 2022 FlexLB Project.
*/

package controllers

import (
	"context"

	"gitee.com/flexlb/flexlb-kube-controller/handlers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// FlexLBClusterReconciler reconciles a FlexLBCluster object
type NodeReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ChangeHandler func(client.Client, context.Context, *v1.Node) error
}

//+kubebuilder:rbac:groups="",resources=node,verbs=get;list;watch;update;patch

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var node v1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.ChangeHandler(r.Client, ctx, &node)
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// new create, add traffic node ip annotation
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			annotations := e.ObjectNew.GetAnnotations()
			_, exist := annotations[handlers.NodeNetworkKey]
			return !exist
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Node{}, builder.WithPredicates(p)).
		Complete(r)
}
