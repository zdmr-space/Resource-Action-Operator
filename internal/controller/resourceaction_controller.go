package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	"de.yusaozdemir.resource-action-operator/internal/engine"
)

// ResourceActionReconciler reconciles a ResourceAction object
type ResourceActionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Engine *engine.Engine
}

// RBAC
// +kubebuilder:rbac:groups=ops.yusaozdemir.de,resources=resourceactions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ops.yusaozdemir.de,resources=resourceactions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ops.yusaozdemir.de,resources=resourceactions/finalizers,verbs=update

func (r *ResourceActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ra opsv1alpha1.ResourceAction
	if err := r.Get(ctx, req.NamespacedName, &ra); err != nil {
		// Objekt gelöscht → nichts zu tun
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Group kann leer sein (core)
	gvk := schema.GroupVersionKind{
		Group:   ra.Spec.Selector.Group,
		Version: ra.Spec.Selector.Version,
		Kind:    ra.Spec.Selector.Kind,
	}

	logger.Info("Ensuring watch for resource",
		"resourceAction", ra.Name,
		"gvk", gvk.String(),
	)

	// Engine anweisen, diese Ressource zu beobachten
	if err := r.Engine.EnsureWatching(ctx, gvk); err != nil {
		logger.Error(err, "failed to ensure watching resource", "gvk", gvk.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opsv1alpha1.ResourceAction{}).
		Named("resourceaction").
		Complete(r)
}
