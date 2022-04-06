/*
Copyright 2022 FlexLB Project.
*/

package controllers

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	crdv1 "gitee.com/flexlb/flexlb-kube-controller/api/v1"
	"gitee.com/flexlb/flexlb-kube-controller/utils"
)

// FlexLBInstanceReconciler reconciles a FlexLBInstance object
type FlexLBInstanceReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	RefreshInterval time.Duration
	ChangeHandler   func(client.Client, context.Context, *crdv1.FlexLBInstance) error
	DeleteHandler   func(client.Client, context.Context, *crdv1.FlexLBInstance) error
}

//+kubebuilder:rbac:groups=crd.flexlb.gitee.io,resources=flexlbinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.flexlb.gitee.io,resources=flexlbinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.flexlb.gitee.io,resources=flexlbinstances/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;

func (r *FlexLBInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var instance crdv1.FlexLBInstance
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{}, nil
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// set finalizer if not set
		if utils.SetFinalizer(&instance.ObjectMeta) {
			log.Log.Info("set finalizer ", "controller", "FlexLBInstanceReconciler", "object", req.NamespacedName)
			if err := r.Update(ctx, &instance); err != nil {
				log.Log.Info("set finalizer failed", "controller", "FlexLBInstanceReconciler", "object", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}

		// process change request
		if err := r.ChangeHandler(r.Client, ctx, &instance); err != nil {
			return ctrl.Result{}, err
		}

	} else {
		// process delete request
		if err := r.DeleteHandler(r.Client, ctx, &instance); err != nil {
			return ctrl.Result{}, err
		}

		// unset finalizer if set
		if utils.UnsetFinalizer(&instance.ObjectMeta) {
			log.Log.Info("unset finalizer ", "controller", "FlexLBInstanceReconciler", "object", req.NamespacedName)
			if err := r.Update(ctx, &instance); err != nil {
				log.Log.Info("unset finalizer failed", "controller", "FlexLBInstanceReconciler", "object", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: r.RefreshInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlexLBInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*crdv1.FlexLBInstance)
			new := e.ObjectNew.(*crdv1.FlexLBInstance)
			// reconcile when spec changed or delete timestamp is set
			return !cmp.Equal(new.Spec, old.Spec) || !new.DeletionTimestamp.IsZero()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1.FlexLBInstance{}, builder.WithPredicates(p)).
		Complete(r)
}
