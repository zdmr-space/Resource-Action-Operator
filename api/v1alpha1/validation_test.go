package v1alpha1

import "testing"

func TestValidateResourceActionSpec_Valid(t *testing.T) {
	spec := ResourceActionSpec{
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
				Mode: "once",
				URLPolicy: &URLPolicySpec{
					AllowedHostRegex: []string{`^api\.example\.com$`},
				},
			},
		},
	}
	if err := ValidateResourceActionSpec(spec); err != nil {
		t.Fatalf("expected valid spec, got error: %v", err)
	}
}

func TestValidateResourceActionSpec_InvalidURLPolicyRegex(t *testing.T) {
	spec := ResourceActionSpec{
		Selector: ResourceSelector{
			Version: "v1",
			Kind:    "Namespace",
		},
		Events: []string{"Create"},
		Actions: []ActionSpec{
			{
				Type: "http",
				URL:  "https://example.com",
				URLPolicy: &URLPolicySpec{
					AllowedHostRegex: []string{"["},
				},
			},
		},
	}
	if err := ValidateResourceActionSpec(spec); err == nil {
		t.Fatalf("expected invalid regex error, got nil")
	}
}

func TestValidateResourceActionSpec_CronRequiresDuration(t *testing.T) {
	spec := ResourceActionSpec{
		Selector: ResourceSelector{
			Version: "v1",
			Kind:    "Namespace",
		},
		Events: []string{"Create"},
		Actions: []ActionSpec{
			{
				Type:     "http",
				URL:      "https://example.com",
				Mode:     "cron",
				Schedule: "not-a-duration",
			},
		},
	}
	if err := ValidateResourceActionSpec(spec); err == nil {
		t.Fatalf("expected invalid schedule error, got nil")
	}
}

func TestValidateResourceActionSpec_ValidJobAction(t *testing.T) {
	allowRunAsRoot := true
	spec := ResourceActionSpec{
		Selector: ResourceSelector{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		},
		Events: []string{"Create"},
		Actions: []ActionSpec{
			{
				Type: "job",
				Job: &JobSpec{
					Image:              "bash:5.2",
					Script:             "echo hello",
					InterpreterCommand: []string{"/bin/bash", "-c"},
					AllowRunAsRoot:     &allowRunAsRoot,
					Env: []JobEnvVar{
						{
							Name:  "STATIC",
							Value: "value",
						},
					},
				},
			},
		},
	}

	if err := ValidateResourceActionSpec(spec); err != nil {
		t.Fatalf("expected valid job spec, got error: %v", err)
	}
}

func TestValidateResourceActionSpec_JobRequiresSingleExecutionMode(t *testing.T) {
	spec := ResourceActionSpec{
		Selector: ResourceSelector{
			Version: "v1",
			Kind:    "Namespace",
		},
		Events: []string{"Create"},
		Actions: []ActionSpec{
			{
				Type: "job",
				Job: &JobSpec{
					Image:   "bash:5.2",
					Script:  "echo hello",
					Command: []string{"/bin/bash", "-c"},
				},
			},
		},
	}

	if err := ValidateResourceActionSpec(spec); err == nil {
		t.Fatalf("expected invalid job configuration, got nil")
	}
}

func TestValidateResourceActionSpec_JobVolumeMountRequiresDefinedVolume(t *testing.T) {
	spec := ResourceActionSpec{
		Selector: ResourceSelector{
			Version: "v1",
			Kind:    "Namespace",
		},
		Events: []string{"Create"},
		Actions: []ActionSpec{
			{
				Type: "job",
				Job: &JobSpec{
					Image:  "bash:5.2",
					Script: "echo hello",
					VolumeMounts: []JobVolumeMount{
						{
							Name:      "missing",
							MountPath: "/var/run/data",
						},
					},
				},
			},
		},
	}

	if err := ValidateResourceActionSpec(spec); err == nil {
		t.Fatalf("expected invalid volume mount reference, got nil")
	}
}
