package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type K8sExecutor struct {
	Client    client.Client
	Clientset kubernetes.Interface
	Recorder  record.EventRecorder
}

func NewK8sExecutor(c client.Client, clientset kubernetes.Interface, recorder ...record.EventRecorder) *K8sExecutor {
	exec := &K8sExecutor{Client: c, Clientset: clientset}
	if len(recorder) > 0 {
		exec.Recorder = recorder[0]
	}
	return exec
}

func (e *K8sExecutor) Execute(ctx context.Context, input MatchInput) error {
	logger := log.FromContext(ctx)

	var list opsv1alpha1.ResourceActionList
	if err := e.Client.List(ctx, &list); err != nil {
		return err
	}

	for _, ra := range list.Items {
		var execErr error
		executedAny := false
		executedActions := 0
		totalAttempts := 0
		totalNetworkRetries := 0
		totalStatusRetries := 0
		totalBackoffMillis := int64(0)
		totalDurationMillis := int64(0)
		lastHTTPStatus := 0
		var lastJobDetails *opsv1alpha1.JobExecutionRecord

		if !matchesSelector(ra.Spec.Selector, input.GVK) {
			continue
		}
		if !containsEvent(ra.Spec.Events, string(input.Event)) {
			continue
		}
		if !matchesFilters(ra.Spec.Filters, input) {
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
		jobExec := NewJobExecutor(e.Client, e.Clientset)

		for i, action := range ra.Spec.Actions {
			if action.Mode == "cron" || action.Mode == "schedule" {
				continue
			}
			executedAny = true

			logger.Info("Executing action",
				"resourceAction", ra.Name,
				"actionIndex", i,
				"type", action.Type,
				"event", input.Event,
				"name", input.Obj.GetName(),
			)

			actionMetrics, err := e.executeAction(ctx, ra, i, action, input, httpExec, jobExec)
			totalAttempts += actionMetrics.Attempts
			totalNetworkRetries += actionMetrics.NetworkRetryCount
			totalStatusRetries += actionMetrics.StatusRetryCount
			totalBackoffMillis += actionMetrics.BackoffMillis
			totalDurationMillis += actionMetrics.DurationMillis
			if actionMetrics.StatusCode > 0 {
				lastHTTPStatus = actionMetrics.StatusCode
			}
			if actionMetrics.Job != nil {
				lastJobDetails = actionMetrics.Job.DeepCopy()
			}
			executedActions++
			if err != nil {
				execErr = err
				break
			}
		}
		if !executedAny {
			continue
		}

		// ---- Status Update (CONFLICT-SAFE) ----
		execRecord := opsv1alpha1.ExecutionRecord{
			ResourceUID:       string(input.Obj.GetUID()),
			Event:             string(input.Event),
			ExecutedAt:        metav1.Now(),
			ActionCount:       executedActions,
			Attempts:          totalAttempts,
			RetryCount:        totalNetworkRetries + totalStatusRetries,
			NetworkRetryCount: totalNetworkRetries,
			StatusRetryCount:  totalStatusRetries,
			BackoffMillis:     totalBackoffMillis,
			DurationMillis:    totalDurationMillis,
			LastHTTPStatus:    lastHTTPStatus,
			Job:               lastJobDetails,
		}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var latest opsv1alpha1.ResourceAction
			if err := e.Client.Get(ctx, client.ObjectKey{
				Name:      ra.Name,
				Namespace: ra.Namespace,
			}, &latest); err != nil {
				return err
			}

			latest.Status.Executions = append(latest.Status.Executions, execRecord)

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

		if execErr != nil && executedActions > 0 {
			observeHTTPExecution("failure", HTTPExecutionRecordMetrics{
				ActionCount:       executedActions,
				Attempts:          totalAttempts,
				NetworkRetryCount: totalNetworkRetries,
				StatusRetryCount:  totalStatusRetries,
				BackoffMillis:     totalBackoffMillis,
				DurationMillis:    totalDurationMillis,
				LastHTTPStatus:    lastHTTPStatus,
			})
			e.emitEvent(&ra, corev1.EventTypeWarning, "ActionFailed", execRecord, execErr)
			return execErr
		}

		if totalAttempts > 0 || lastHTTPStatus > 0 || totalDurationMillis > 0 {
			observeHTTPExecution("success", HTTPExecutionRecordMetrics{
				ActionCount:       executedActions,
				Attempts:          totalAttempts,
				NetworkRetryCount: totalNetworkRetries,
				StatusRetryCount:  totalStatusRetries,
				BackoffMillis:     totalBackoffMillis,
				DurationMillis:    totalDurationMillis,
				LastHTTPStatus:    lastHTTPStatus,
			})
		}
		e.emitEvent(&ra, corev1.EventTypeNormal, "ActionSucceeded", execRecord, nil)
	}

	return nil
}

func (e *K8sExecutor) executeAction(
	ctx context.Context,
	ra opsv1alpha1.ResourceAction,
	actionIndex int,
	action opsv1alpha1.ActionSpec,
	input MatchInput,
	httpExec *HTTPExecutor,
	jobExec *JobExecutor,
) (HTTPExecutionMetrics, error) {
	switch action.Type {
	case "http":
		headersResolved, err := e.resolveHeaders(ctx, action.Headers, ra.Namespace)
		if err != nil {
			return HTTPExecutionMetrics{}, err
		}

		return httpExec.ExecuteWithMetrics(ctx, action, ra.Namespace, input.Obj, headersResolved)
	case "job":
		jobMetrics, err := jobExec.Execute(ctx, ra, actionIndex, action, input)
		return HTTPExecutionMetrics{
			Attempts:       jobMetrics.Attempts,
			DurationMillis: jobMetrics.DurationMillis,
			Job:            jobMetrics.Details,
		}, err
	default:
		return HTTPExecutionMetrics{}, fmt.Errorf("unsupported action type %q", action.Type)
	}
}

func (e *K8sExecutor) emitEvent(
	ra *opsv1alpha1.ResourceAction,
	eventType string,
	reason string,
	execRecord opsv1alpha1.ExecutionRecord,
	execErr error,
) {
	if e.Recorder == nil {
		return
	}

	msg := fmt.Sprintf(
		"event=%s actions=%d attempts=%d retries=%d networkRetries=%d statusRetries=%d backoffMs=%d durationMs=%d status=%d",
		execRecord.Event,
		execRecord.ActionCount,
		execRecord.Attempts,
		execRecord.RetryCount,
		execRecord.NetworkRetryCount,
		execRecord.StatusRetryCount,
		execRecord.BackoffMillis,
		execRecord.DurationMillis,
		execRecord.LastHTTPStatus,
	)
	if execErr != nil {
		msg = fmt.Sprintf("%s error=%v", msg, execErr)
	}

	e.Recorder.Event(ra, eventType, reason, msg)
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

func matchesFilters(filter *opsv1alpha1.FilterSpec, input MatchInput) bool {
	if filter == nil {
		return true
	}
	obj := input.Obj

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

	if len(filter.LabelChanges) > 0 {
		if input.Event != EventUpdate || input.OldObj == nil {
			return false
		}
		oldLabels := input.OldObj.GetLabels()
		newLabels := obj.GetLabels()
		for _, change := range filter.LabelChanges {
			if !matchesLabelChange(change, oldLabels, newLabels) {
				return false
			}
		}
	}

	return true
}

func matchesLabelChange(change opsv1alpha1.LabelChangeFilter, oldLabels, newLabels map[string]string) bool {
	oldValue, oldExists := oldLabels[change.Key]
	newValue, newExists := newLabels[change.Key]

	return labelValueMatches(change.From, oldExists, oldValue) &&
		labelValueMatches(change.To, newExists, newValue)
}

func labelValueMatches(expected string, exists bool, value string) bool {
	switch expected {
	case "":
		return !exists
	case "*":
		return exists
	default:
		return exists && value == expected
	}
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

			// Update transition time only when status changes.
			if existing.Status != cond.Status {
				cond.LastTransitionTime = now
			} else {
				cond.LastTransitionTime = existing.LastTransitionTime
			}

			ra.Status.Conditions[i] = cond
			return
		}
	}

	// Append new condition.
	ra.Status.Conditions = append(ra.Status.Conditions, cond)
}
