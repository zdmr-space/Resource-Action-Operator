package v1alpha1

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ValidateResourceActionSpec performs runtime-safe validation for fields that
// are difficult to express completely in CRD schema markers.
func ValidateResourceActionSpec(spec ResourceActionSpec) error {
	if spec.Selector.Version == "" || spec.Selector.Kind == "" {
		return fmt.Errorf("selector.version and selector.kind are required")
	}
	if len(spec.Events) == 0 {
		return fmt.Errorf("at least one event is required")
	}
	if len(spec.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	if spec.Filters != nil {
		if spec.Filters.NameRegex != "" {
			if _, err := regexp.Compile(spec.Filters.NameRegex); err != nil {
				return fmt.Errorf("invalid filters.nameRegex: %w", err)
			}
		}
		if spec.Filters.NamespaceRegex != "" {
			if _, err := regexp.Compile(spec.Filters.NamespaceRegex); err != nil {
				return fmt.Errorf("invalid filters.namespaceRegex: %w", err)
			}
		}
	}

	for i, action := range spec.Actions {
		if action.Mode == "cron" || action.Mode == "schedule" {
			if action.Schedule == "" {
				return fmt.Errorf("actions[%d].schedule is required for mode %q", i, action.Mode)
			}
			if _, err := time.ParseDuration(action.Schedule); err != nil {
				return fmt.Errorf("actions[%d].schedule invalid duration: %w", i, err)
			}
		}
		switch action.Type {
		case "http":
			if err := validateHTTPAction(i, action); err != nil {
				return err
			}
		case "job":
			if err := validateJobAction(i, action); err != nil {
				return err
			}
		default:
			return fmt.Errorf("actions[%d].type must be \"http\" or \"job\"", i)
		}
	}

	return nil
}

func validateHTTPAction(i int, action ActionSpec) error {
	if action.Job != nil {
		return fmt.Errorf("actions[%d].job is only allowed for type %q", i, action.Type)
	}
	if action.URL == "" {
		return fmt.Errorf("actions[%d].url is required", i)
	}
	if err := validateActionURL(action.URL); err != nil {
		return fmt.Errorf("actions[%d].url: %w", i, err)
	}
	if action.ExpectedStatus != "" {
		if _, err := regexp.Compile(action.ExpectedStatus); err != nil {
			return fmt.Errorf("actions[%d].expectedStatus invalid regex: %w", i, err)
		}
	}
	if action.URLPolicy != nil {
		for _, p := range action.URLPolicy.AllowedHostRegex {
			if _, err := regexp.Compile(p); err != nil {
				return fmt.Errorf("actions[%d].urlPolicy.allowedHostRegex invalid regex %q: %w", i, p, err)
			}
		}
		for _, p := range action.URLPolicy.BlockedHostRegex {
			if _, err := regexp.Compile(p); err != nil {
				return fmt.Errorf("actions[%d].urlPolicy.blockedHostRegex invalid regex %q: %w", i, p, err)
			}
		}
	}
	return nil
}

func validateJobAction(i int, action ActionSpec) error {
	if action.Job == nil {
		return fmt.Errorf("actions[%d].job is required for type %q", i, action.Type)
	}
	if action.URL != "" {
		return fmt.Errorf("actions[%d].url is only allowed for type %q", i, action.Type)
	}

	job := action.Job
	if strings.TrimSpace(job.Image) == "" {
		return fmt.Errorf("actions[%d].job.image is required", i)
	}
	if err := validateJobExecution(i, job); err != nil {
		return err
	}
	if err := validateJobEnv(i, job); err != nil {
		return err
	}
	volumesByName, err := validateJobVolumes(i, job)
	if err != nil {
		return err
	}
	if err := validateJobVolumeMounts(i, job, volumesByName); err != nil {
		return err
	}
	if job.Timeout != "" {
		if _, parseErr := time.ParseDuration(job.Timeout); parseErr != nil {
			return fmt.Errorf("actions[%d].job.timeout invalid duration: %w", i, parseErr)
		}
	}
	return nil
}

func validateJobExecution(i int, job *JobSpec) error {
	hasScript := strings.TrimSpace(job.Script) != ""
	hasCommand := len(job.Command) > 0
	if hasScript == hasCommand {
		return fmt.Errorf("actions[%d].job must define exactly one of script or command", i)
	}
	if hasScript && len(job.Args) > 0 {
		return fmt.Errorf("actions[%d].job.args is not supported when script is set", i)
	}
	return validateNonEmptyStrings(i, "job.command", job.Command)
}

func validateJobEnv(i int, job *JobSpec) error {
	if err := validateNonEmptyStrings(i, "job.args", job.Args); err != nil {
		return err
	}
	if err := validateNonEmptyStrings(i, "job.interpreterCommand", job.InterpreterCommand); err != nil {
		return err
	}

	for j, env := range job.Env {
		if strings.TrimSpace(env.Name) == "" {
			return fmt.Errorf("actions[%d].job.env[%d].name is required", i, j)
		}
		hasValue := env.Value != ""
		hasValueFrom := env.ValueFrom != nil
		if hasValue == hasValueFrom {
			return fmt.Errorf("actions[%d].job.env[%d] must define exactly one of value or valueFrom", i, j)
		}
		if hasValueFrom && env.ValueFrom.SecretKeyRef == nil {
			return fmt.Errorf("actions[%d].job.env[%d].valueFrom.secretKeyRef is required", i, j)
		}
	}

	return nil
}

func validateJobVolumes(i int, job *JobSpec) (map[string]struct{}, error) {
	volumesByName := make(map[string]struct{}, len(job.Volumes))
	for j, vol := range job.Volumes {
		if strings.TrimSpace(vol.Name) == "" {
			return nil, fmt.Errorf("actions[%d].job.volumes[%d].name is required", i, j)
		}
		if _, exists := volumesByName[vol.Name]; exists {
			return nil, fmt.Errorf("actions[%d].job.volumes[%d].name %q is duplicated", i, j, vol.Name)
		}
		volumesByName[vol.Name] = struct{}{}
		hasSecret := vol.Secret != nil
		hasConfigMap := vol.ConfigMap != nil
		if hasSecret == hasConfigMap {
			return nil, fmt.Errorf("actions[%d].job.volumes[%d] must define exactly one of secret or configMap", i, j)
		}
		if hasSecret && strings.TrimSpace(vol.Secret.SecretName) == "" {
			return nil, fmt.Errorf("actions[%d].job.volumes[%d].secret.secretName is required", i, j)
		}
		if hasConfigMap && strings.TrimSpace(vol.ConfigMap.Name) == "" {
			return nil, fmt.Errorf("actions[%d].job.volumes[%d].configMap.name is required", i, j)
		}
	}

	return volumesByName, nil
}

func validateJobVolumeMounts(i int, job *JobSpec, volumesByName map[string]struct{}) error {
	for j, mount := range job.VolumeMounts {
		if strings.TrimSpace(mount.Name) == "" {
			return fmt.Errorf("actions[%d].job.volumeMounts[%d].name is required", i, j)
		}
		if strings.TrimSpace(mount.MountPath) == "" {
			return fmt.Errorf("actions[%d].job.volumeMounts[%d].mountPath is required", i, j)
		}
		if _, exists := volumesByName[mount.Name]; !exists {
			return fmt.Errorf("actions[%d].job.volumeMounts[%d].name %q does not reference a defined volume", i, j, mount.Name)
		}
	}

	return nil
}

func validateNonEmptyStrings(i int, field string, values []string) error {
	for j, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("actions[%d].%s[%d] must not be empty", i, field, j)
		}
	}

	return nil
}

func validateActionURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if strings.TrimSpace(u.Hostname()) == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}
