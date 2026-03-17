package v1alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-ops-yusaozdemir-de-v1alpha1-resourceaction,mutating=false,failurePolicy=Fail,sideEffects=None,groups=ops.yusaozdemir.de,resources=resourceactions,verbs=create;update,versions=v1alpha1,name=vresourceaction.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &ResourceActionCustomValidator{}

type ResourceActionCustomValidator struct{}

func (r *ResourceAction) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&ResourceActionCustomValidator{}).
		Complete()
}

func (v *ResourceActionCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	ra, ok := obj.(*ResourceAction)
	if !ok {
		return nil, fmt.Errorf("expected a ResourceAction object but got %T", obj)
	}
	return nil, validateResourceActionObject(ra)
}

func (v *ResourceActionCustomValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	ra, ok := newObj.(*ResourceAction)
	if !ok {
		return nil, fmt.Errorf("expected a ResourceAction object but got %T", newObj)
	}
	return nil, validateResourceActionObject(ra)
}

func (v *ResourceActionCustomValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateResourceActionObject(ra *ResourceAction) error {
	if err := ValidateResourceActionSpec(ra.Spec); err != nil {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "ResourceAction"},
			ra.Name,
			field.ErrorList{
				field.Invalid(field.NewPath("spec"), ra.Spec, fmt.Sprintf("invalid ResourceAction spec: %v", err)),
			},
		)
	}
	return nil
}
