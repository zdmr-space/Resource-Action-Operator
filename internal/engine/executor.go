package engine

import (
	"context"
	"regexp"
	"strings"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type K8sExecutor struct {
	Client client.Client
}

func NewK8sExecutor(c client.Client) *K8sExecutor {
	return &K8sExecutor{Client: c}
}

func (e *K8sExecutor) Execute(ctx context.Context, input MatchInput) error {
	logger := log.FromContext(ctx)

	var list opsv1alpha1.ResourceActionList
	if err := e.Client.List(ctx, &list); err != nil {
		return err
	}

	for _, ra := range list.Items {
		var execErr error

		if !matchesSelector(ra.Spec.Selector, input.GVK) {
			continue
		}
		if !containsEvent(ra.Spec.Events, string(input.Event)) {
			continue
		}
		if !matchesFilters(ra.Spec.Filters, input.Obj) {
			continue
		}
		if alreadyExecuted(&ra, input.Obj.GetUID(), string(input.Event)) {
			logger.Info("Skipping already executed action",
				"resourceAction", ra.Name,
				"event", input.Event,
				"name", input.Obj.GetName(),
			)
			continue
		}

		httpExec := NewHTTPExecutor(e.Client)

		for i, action := range ra.Spec.Actions {

			if action.Mode == "cron" {
				continue
			}
			if action.Type != "http" {
				continue
			}

			logger.Info("Executing action",
				"resourceAction", ra.Name,
				"actionIndex", i,
				"type", action.Type,
				"event", input.Event,
				"name", input.Obj.GetName(),
			)

			headersResolved, err := e.resolveHeaders(ctx, action.Headers, ra.Namespace)
			if err != nil {
				execErr = err
				break
			}

			if err := httpExec.Execute(ctx, action, ra.Namespace, input.Obj, headersResolved); err != nil {
				execErr = err
				break
			}
		}

		// ---- Status Update (CONFLICT-SAFE) ----
		record := opsv1alpha1.ExecutionRecord{
			ResourceUID: string(input.Obj.GetUID()),
			Event:       string(input.Event),
			ExecutedAt:  metav1.Now(),
		}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var latest opsv1alpha1.ResourceAction
			if err := e.Client.Get(ctx, client.ObjectKey{
				Name:      ra.Name,
				Namespace: ra.Namespace,
			}, &latest); err != nil {
				return err
			}

			latest.Status.Executions = append(latest.Status.Executions, record)

			if execErr != nil {
				latest.Status.LastError = execErr.Error()
				setCondition(&latest, metav1.Condition{
					Type:    "Ready",
					Status:  metav1.ConditionFalse,
					Reason:  "ActionFailed",
					Message: execErr.Error(),
				})
			} else {
				latest.Status.LastError = ""
				setCondition(&latest, metav1.Condition{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "ActionSucceeded",
					Message: "All actions executed successfully",
				})
			}

			return e.Client.Status().Update(ctx, &latest)
		})

		if err != nil {
			logger.Error(err, "failed to update status", "resourceAction", ra.Name)
			return err
		}

		if execErr != nil {
			return execErr
		}
	}

	return nil
}

func (e *K8sExecutor) resolveHeaders(
	ctx context.Context,
	headers map[string]opsv1alpha1.ValueFrom,
	namespace string,
) (map[string]string, error) {

	resolved := make(map[string]string)

	for key, val := range headers {
		if val.SecretKeyRef != nil {
			var secret corev1.Secret
			if err := e.Client.Get(ctx, client.ObjectKey{
				Name:      val.SecretKeyRef.Name,
				Namespace: namespace,
			}, &secret); err != nil {
				return nil, err
			}

			resolved[key] = string(secret.Data[val.SecretKeyRef.Key])
		}
	}

	return resolved, nil
}

func alreadyExecuted(
	ra *opsv1alpha1.ResourceAction,
	uid types.UID,
	event string,
) bool {
	for _, exec := range ra.Status.Executions {
		if exec.ResourceUID == string(uid) && exec.Event == event {
			return true
		}
	}
	return false
}

func matchesSelector(sel opsv1alpha1.ResourceSelector, gvk schema.GroupVersionKind) bool {
	return sel.Group == gvk.Group &&
		sel.Version == gvk.Version &&
		sel.Kind == gvk.Kind
}

func matchesFilters(filter *opsv1alpha1.FilterSpec, obj *unstructured.Unstructured) bool {
	if filter == nil {
		return true
	}

	if filter.NameRegex != "" {
		re, err := regexp.Compile(filter.NameRegex)
		if err != nil || !re.MatchString(obj.GetName()) {
			return false
		}
	}

	if filter.NamespaceRegex != "" {
		re, err := regexp.Compile(filter.NamespaceRegex)
		if err != nil || !re.MatchString(obj.GetNamespace()) {
			return false
		}
	}

	if len(filter.Labels) > 0 {
		labels := obj.GetLabels()
		for k, v := range filter.Labels {
			if labels[k] != v {
				return false
			}
		}
	}

	return true
}

func containsEvent(events []string, ev string) bool {
	for _, e := range events {
		if strings.EqualFold(e, ev) {
			return true
		}
	}
	return false
}

func setCondition(
	ra *opsv1alpha1.ResourceAction,
	cond metav1.Condition,
) {
	now := metav1.Now()

	cond.ObservedGeneration = ra.Generation

	if cond.LastTransitionTime.IsZero() {
		cond.LastTransitionTime = now
	}

	if ra.Status.Conditions == nil {
		ra.Status.Conditions = []metav1.Condition{}
	}

	for i, existing := range ra.Status.Conditions {
		if existing.Type == cond.Type {

			// Nur Transition-Zeit ändern, wenn sich Status ändert
			if existing.Status != cond.Status {
				cond.LastTransitionTime = now
			} else {
				cond.LastTransitionTime = existing.LastTransitionTime
			}

			ra.Status.Conditions[i] = cond
			return
		}
	}

	// Neue Condition
	ra.Status.Conditions = append(ra.Status.Conditions, cond)
}
