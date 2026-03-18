package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JobExecutionMetrics struct {
	Attempts       int
	DurationMillis int64
}

type JobExecutor struct {
	k8s client.Client
}

func NewJobExecutor(k8s client.Client) *JobExecutor {
	return &JobExecutor{k8s: k8s}
}

func (e *JobExecutor) Execute(
	ctx context.Context,
	ra opsv1alpha1.ResourceAction,
	actionIndex int,
	action opsv1alpha1.ActionSpec,
	input MatchInput,
) (JobExecutionMetrics, error) {
	startedAt := time.Now()
	metrics := JobExecutionMetrics{Attempts: 1}

	if action.Job == nil {
		return metrics, fmt.Errorf("job action is missing spec")
	}

	jobObj, err := buildJobForAction(ra, actionIndex, action, input)
	if err != nil {
		return metrics, err
	}

	if err := e.k8s.Create(ctx, jobObj); err != nil {
		return metrics, err
	}

	metrics.DurationMillis = time.Since(startedAt).Milliseconds()
	return metrics, nil
}

func buildJobForAction(
	ra opsv1alpha1.ResourceAction,
	actionIndex int,
	action opsv1alpha1.ActionSpec,
	input MatchInput,
) (*batchv1.Job, error) {
	job := action.Job
	command, args, err := resolveJobCommand(job)
	if err != nil {
		return nil, err
	}

	envVars := make([]corev1.EnvVar, 0, len(job.Env))
	for _, item := range job.Env {
		envVar := corev1.EnvVar{Name: item.Name}
		if item.ValueFrom != nil && item.ValueFrom.SecretKeyRef != nil {
			envVar.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: item.ValueFrom.SecretKeyRef.Name},
					Key:                  item.ValueFrom.SecretKeyRef.Key,
				},
			}
		} else {
			envVar.Value = item.Value
		}
		envVars = append(envVars, envVar)
	}

	volumes := make([]corev1.Volume, 0, len(job.Volumes))
	for _, item := range job.Volumes {
		volume := corev1.Volume{Name: item.Name}
		switch {
		case item.Secret != nil:
			volume.VolumeSource = corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: item.Secret.SecretName,
				},
			}
		case item.ConfigMap != nil:
			volume.VolumeSource = corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: item.ConfigMap.Name},
				},
			}
		}
		volumes = append(volumes, volume)
	}

	volumeMounts := make([]corev1.VolumeMount, 0, len(job.VolumeMounts))
	for _, item := range job.VolumeMounts {
		readOnly := true
		if item.ReadOnly != nil {
			readOnly = *item.ReadOnly
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      item.Name,
			MountPath: item.MountPath,
			ReadOnly:  readOnly,
		})
	}

	timeout := int64(parseDurationDefault(job.Timeout, 30*time.Second).Seconds())
	if timeout <= 0 {
		timeout = 30
	}

	backoffLimit := int32(0)
	if job.BackoffLimit != nil {
		backoffLimit = *job.BackoffLimit
	}

	automount := false
	if job.AutomountServiceAccountToken != nil {
		automount = *job.AutomountServiceAccountToken
	}

	allowPrivilegeEscalation := false
	runAsNonRoot := job.AllowRunAsRoot == nil || !*job.AllowRunAsRoot
	readOnlyRootFilesystem := false
	container := corev1.Container{
		Name:            "runner",
		Image:           job.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         command,
		Args:            args,
		Env:             envVars,
		VolumeMounts:    volumeMounts,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
			RunAsNonRoot:             &runAsNonRoot,
			ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}
	if job.Resources != nil {
		container.Resources = *job.Resources.DeepCopy()
	}

	podSpec := corev1.PodSpec{
		RestartPolicy:                 corev1.RestartPolicyNever,
		AutomountServiceAccountToken:  &automount,
		ServiceAccountName:            job.ServiceAccountName,
		Containers:                    []corev1.Container{container},
		Volumes:                       volumes,
		EnableServiceLinks:            ptrTo(false),
		TerminationGracePeriodSeconds: ptrTo(int64(0)),
	}

	jobObj := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ra.Namespace,
			GenerateName: jobGenerateName(ra.Name, actionIndex),
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":                  "resource-action-operator",
				"resource-action-operator.yusaozdemir.de/name":  ra.Name,
				"resource-action-operator.yusaozdemir.de/type":  action.Type,
				"resource-action-operator.yusaozdemir.de/event": strings.ToLower(string(input.Event)),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: job.TTLSecondsAfterFinished,
			ActiveDeadlineSeconds:   &timeout,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"resource-action-operator.yusaozdemir.de/name": ra.Name,
					},
				},
				Spec: podSpec,
			},
		},
	}

	return jobObj, nil
}

func resolveJobCommand(job *opsv1alpha1.JobSpec) ([]string, []string, error) {
	if job == nil {
		return nil, nil, fmt.Errorf("job spec is required")
	}
	if strings.TrimSpace(job.Script) != "" {
		interpreter := job.InterpreterCommand
		if len(interpreter) == 0 {
			interpreter = []string{"/bin/sh", "-c"}
		}
		return interpreter, []string{job.Script}, nil
	}
	if len(job.Command) == 0 {
		return nil, nil, fmt.Errorf("job command is required")
	}
	return job.Command, job.Args, nil
}

func jobGenerateName(resourceActionName string, actionIndex int) string {
	name := strings.ToLower(resourceActionName)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.Trim(name, "-")
	if len(name) > 36 {
		name = name[:36]
	}
	if name == "" {
		name = "resource-action"
	}
	return fmt.Sprintf("%s-a%d-%s-", name, actionIndex, rand.String(5))
}

func ptrTo[T any](value T) *T {
	return &value
}
