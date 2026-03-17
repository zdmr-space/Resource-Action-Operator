/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
)

type noopEnsurer struct{}

func (n *noopEnsurer) EnsureWatching(_ context.Context, _ schema.GroupVersionKind) error {
	return nil
}

var _ = Describe("ResourceAction Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		resourceaction := &opsv1alpha1.ResourceAction{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ResourceAction")
			err := k8sClient.Get(ctx, typeNamespacedName, resourceaction)
			if err != nil && errors.IsNotFound(err) {
				resource := &opsv1alpha1.ResourceAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: opsv1alpha1.ResourceActionSpec{
						Selector: opsv1alpha1.ResourceSelector{
							Group:   "",
							Version: "v1",
							Kind:    "Namespace",
						},
						Events: []string{"Create"},
						Actions: []opsv1alpha1.ActionSpec{
							{
								Type: "http",
								Mode: "once",
								URL:  "https://example.invalid",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &opsv1alpha1.ResourceAction{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ResourceAction")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ResourceActionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Engine: &noopEnsurer{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should return an error when engine is not configured", func() {
			controllerReconciler := &ResourceActionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("engine is not configured"))
		})

		It("should set SpecValid=False for invalid spec and not fail reconcile", func() {
			invalidName := "invalid-resourceaction"
			invalidKey := types.NamespacedName{Name: invalidName, Namespace: "default"}
			invalid := &opsv1alpha1.ResourceAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      invalidName,
					Namespace: "default",
				},
				Spec: opsv1alpha1.ResourceActionSpec{
					Selector: opsv1alpha1.ResourceSelector{
						Group:   "",
						Version: "v1",
						Kind:    "Namespace",
					},
					Events: []string{"Create"},
					Actions: []opsv1alpha1.ActionSpec{
						{
							Type: "http",
							URL:  "://broken-url",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, invalid)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, invalid)
			}()

			controllerReconciler := &ResourceActionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Engine: &noopEnsurer{},
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: invalidKey,
			})
			Expect(err).NotTo(HaveOccurred())

			var got opsv1alpha1.ResourceAction
			Expect(k8sClient.Get(ctx, invalidKey, &got)).To(Succeed())
			var specValid *metav1.Condition
			for i := range got.Status.Conditions {
				if got.Status.Conditions[i].Type == "SpecValid" {
					specValid = &got.Status.Conditions[i]
					break
				}
			}
			Expect(specValid).NotTo(BeNil())
			Expect(specValid.Status).To(Equal(metav1.ConditionFalse))
			Expect(specValid.Reason).To(Equal("ValidationFailed"))
		})
	})
})
