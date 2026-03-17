package engine

import (
	"context"
	"testing"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestExecutor(t *testing.T, objects ...client.Object) (*K8sExecutor, client.Client) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := opsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add batch scheme: %v", err)
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&opsv1alpha1.ResourceAction{}).
		WithObjects(objects...).
		Build()
	return NewK8sExecutor(cl), cl
}

func newDeploymentInput(uid, name, namespace string) MatchInput {
	return MatchInput{
		Event: EventCreate,
		GVK: schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		},
		Obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": namespace,
					"uid":       uid,
				},
			},
		},
	}
}

func TestExecute_CronOnlyAction_DoesNotWriteStatus(t *testing.T) {
	ra := &opsv1alpha1.ResourceAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ra-cron-only",
			Namespace: "default",
		},
		Spec: opsv1alpha1.ResourceActionSpec{
			Selector: opsv1alpha1.ResourceSelector{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			Events: []string{"Create"},
			Actions: []opsv1alpha1.ActionSpec{
				{
					Type:     "http",
					Mode:     "cron",
					Schedule: "30s",
					URL:      "http://example.invalid",
				},
			},
		},
	}

	exec, cl := newTestExecutor(t, ra)
	input := newDeploymentInput("uid-1", "demo", "default")

	if err := exec.Execute(context.Background(), input); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var got opsv1alpha1.ResourceAction
	if err := cl.Get(context.Background(), types.NamespacedName{Name: ra.Name, Namespace: ra.Namespace}, &got); err != nil {
		t.Fatalf("get resourceaction: %v", err)
	}
	if len(got.Status.Executions) != 0 {
		t.Fatalf("expected no execution records, got %d", len(got.Status.Executions))
	}
	if len(got.Status.Conditions) != 0 {
		t.Fatalf("expected no conditions, got %d", len(got.Status.Conditions))
	}
}

func TestExecute_ScheduleModeAction_DoesNotWriteStatus(t *testing.T) {
	ra := &opsv1alpha1.ResourceAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ra-schedule-only",
			Namespace: "default",
		},
		Spec: opsv1alpha1.ResourceActionSpec{
			Selector: opsv1alpha1.ResourceSelector{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			Events: []string{"Create"},
			Actions: []opsv1alpha1.ActionSpec{
				{
					Type:     "http",
					Mode:     "schedule",
					Schedule: "30s",
					URL:      "http://example.invalid",
				},
			},
		},
	}

	exec, cl := newTestExecutor(t, ra)
	input := newDeploymentInput("uid-2", "demo2", "default")

	if err := exec.Execute(context.Background(), input); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var got opsv1alpha1.ResourceAction
	if err := cl.Get(context.Background(), types.NamespacedName{Name: ra.Name, Namespace: ra.Namespace}, &got); err != nil {
		t.Fatalf("get resourceaction: %v", err)
	}
	if len(got.Status.Executions) != 0 {
		t.Fatalf("expected no execution records, got %d", len(got.Status.Executions))
	}
	if len(got.Status.Conditions) != 0 {
		t.Fatalf("expected no conditions, got %d", len(got.Status.Conditions))
	}
}

func TestExecute_JobAction_CreatesBatchJob(t *testing.T) {
	ra := &opsv1alpha1.ResourceAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ra-job",
			Namespace: "default",
		},
		Spec: opsv1alpha1.ResourceActionSpec{
			Selector: opsv1alpha1.ResourceSelector{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			Events: []string{"Create"},
			Actions: []opsv1alpha1.ActionSpec{
				{
					Type: "job",
					Job: &opsv1alpha1.JobSpec{
						Image:              "bash:5.2",
						Script:             "echo hello",
						InterpreterCommand: []string{"/bin/bash", "-c"},
						Volumes: []opsv1alpha1.JobVolume{
							{
								Name: "tls",
								Secret: &opsv1alpha1.JobSecretVolume{
									SecretName: "api-client-cert",
								},
							},
							{
								Name: "scripts",
								ConfigMap: &opsv1alpha1.JobConfigMapVolume{
									Name: "job-scripts",
								},
							},
						},
						VolumeMounts: []opsv1alpha1.JobVolumeMount{
							{
								Name:      "tls",
								MountPath: "/var/run/tls",
							},
							{
								Name:      "scripts",
								MountPath: "/opt/scripts",
							},
						},
						ServiceAccountName: "restricted-runner",
					},
				},
			},
		},
	}

	exec, cl := newTestExecutor(t, ra)
	input := newDeploymentInput("uid-3", "demo3", "default")

	if err := exec.Execute(context.Background(), input); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var jobs batchv1.JobList
	if err := cl.List(context.Background(), &jobs); err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs.Items) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs.Items))
	}

	job := jobs.Items[0]
	if job.Spec.Template.Spec.ServiceAccountName != "restricted-runner" {
		t.Fatalf("expected service account %q, got %q", "restricted-runner", job.Spec.Template.Spec.ServiceAccountName)
	}
	if job.Spec.Template.Spec.AutomountServiceAccountToken == nil || *job.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatalf("expected automountServiceAccountToken=false by default")
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(job.Spec.Template.Spec.Containers))
	}
	container := job.Spec.Template.Spec.Containers[0]
	if got := container.Image; got != "bash:5.2" {
		t.Fatalf("expected image bash:5.2, got %q", got)
	}
	if len(container.Command) != 2 || container.Command[0] != "/bin/bash" || container.Command[1] != "-c" {
		t.Fatalf("unexpected command: %#v", container.Command)
	}
	if len(container.Args) != 1 || container.Args[0] != "echo hello" {
		t.Fatalf("unexpected args: %#v", container.Args)
	}
	if len(job.Spec.Template.Spec.Volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(job.Spec.Template.Spec.Volumes))
	}
	if len(container.VolumeMounts) != 2 {
		t.Fatalf("expected 2 volume mounts, got %d", len(container.VolumeMounts))
	}
	if !container.VolumeMounts[0].ReadOnly || !container.VolumeMounts[1].ReadOnly {
		t.Fatalf("expected all volume mounts to be read-only by default")
	}
}
