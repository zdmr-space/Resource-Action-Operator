package v1alpha1

import (
	"context"
	"testing"
)

func TestResourceActionValidateCreate_Valid(t *testing.T) {
	v := &ResourceActionCustomValidator{}
	ra := &ResourceAction{
		Spec: ResourceActionSpec{
			Selector: ResourceSelector{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			Events: []string{"Create"},
			Actions: []ActionSpec{
				{
					Type: "http",
					URL:  "https://api.example.com/hook",
				},
			},
		},
	}
	if _, err := v.ValidateCreate(context.Background(), ra); err != nil {
		t.Fatalf("expected valid create, got error: %v", err)
	}
}

func TestResourceActionValidateCreate_Invalid(t *testing.T) {
	v := &ResourceActionCustomValidator{}
	ra := &ResourceAction{
		Spec: ResourceActionSpec{
			Selector: ResourceSelector{
				Version: "v1",
				Kind:    "Namespace",
			},
			Events: []string{"Create"},
			Actions: []ActionSpec{
				{
					Type: "http",
					URL:  "://bad",
				},
			},
		},
	}
	if _, err := v.ValidateCreate(context.Background(), ra); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}
