/*
Copyright 2022 FlexLB Project.
*/

package controllers

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	crdv1 "gitee.com/flexlb/flexlb-kube-controller/api/v1"
)

// FlexLBClusterReconciler reconciles a FlexLBCluster object
type FlexLBClusterReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Namespace     string
	ChangeHandler func(client.Client, context.Context, *crdv1.FlexLBCluster) error
}

//+kubebuilder:rbac:groups=crd.flexlb.gitee.io,resources=flexlbclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=crd.flexlb.gitee.io,resources=flexlbclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;

func (r *FlexLBClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var cluster crdv1.FlexLBCluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.ChangeHandler(r.Client, ctx, &cluster)
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlexLBClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == r.Namespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.GetNamespace() != r.Namespace {
				return false
			}
			old := e.ObjectOld.(*crdv1.FlexLBCluster)
			new := e.ObjectNew.(*crdv1.FlexLBCluster)
			// reconcile when spec changed or delete timestamp is set
			return !cmp.Equal(new.Spec, old.Spec) || !new.DeletionTimestamp.IsZero()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetNamespace() == r.Namespace
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetNamespace() == r.Namespace
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1.FlexLBCluster{}, builder.WithPredicates(p)).
		Complete(r)
}
