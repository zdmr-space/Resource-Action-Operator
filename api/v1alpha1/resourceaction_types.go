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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ResourceActionSpec defines the desired state of ResourceAction.
type ResourceActionSpec struct {
	Selector ResourceSelector `json:"selector"`
	// +kubebuilder:validation:Items:Enum=Create;Update;Delete
	Events  []string     `json:"events"`
	Filters *FilterSpec  `json:"filters,omitempty"`
	Actions []ActionSpec `json:"actions"`
}

type ResourceSelector struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

type FilterSpec struct {
	Labels         map[string]string `json:"labels,omitempty"`
	NameRegex      string            `json:"nameRegex,omitempty"`
	NamespaceRegex string            `json:"namespaceRegex,omitempty"`
}

type ActionSpec struct {
	// +kubebuilder:validation:Enum=http
	Type string `json:"type"`

	// +kubebuilder:default=POST
	Method  string               `json:"method,omitempty"`
	URL     string               `json:"url,omitempty"`
	Headers map[string]ValueFrom `json:"headers,omitempty"`
	Body    *TemplateSpec        `json:"body,omitempty"`

	ExpectedStatus string `json:"expectedStatus,omitempty"`

	// +kubebuilder:validation:Enum=once;cron
	// +kubebuilder:default=once
	Mode string `json:"mode,omitempty"`

	Schedule string `json:"schedule,omitempty"`

	// +kubebuilder:default="10s"
	Timeout string `json:"timeout,omitempty"`

	Retry *RetrySpec `json:"retry,omitempty"`
	TLS   *TLSSpec   `json:"tls,omitempty"`
}

type RetrySpec struct {
	// +kubebuilder:default=1
	MaxAttempts int `json:"maxAttempts,omitempty"`

	// Basis Backoff, z.B. "500ms"
	// +kubebuilder:default="500ms"
	Backoff string `json:"backoff,omitempty"`

	// Max Backoff, z.B. "10s"
	// +kubebuilder:default="10s"
	MaxBackoff string `json:"maxBackoff,omitempty"`

	// Retry bei Netzwerkfehlern
	// +kubebuilder:default=true
	RetryOnNetworkError *bool `json:"retryOnNetworkError,omitempty"`

	// Statuscodes die retrybar sind
	// +kubebuilder:default:={429,500,502,503,504}
	RetryOnStatus []int `json:"retryOnStatus,omitempty"`
}

type TLSSpec struct {
	// HTTPS verify deaktivieren (dev)
	// +kubebuilder:default=false
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// Optional: SNI/ServerName
	ServerName string `json:"serverName,omitempty"`

	// CA Bundle aus Secret (PEM), default key: ca.crt
	CaSecretRef *SecretKeyRef `json:"caSecretRef,omitempty"`

	// mTLS Client Cert/Key aus Secret, default keys: tls.crt/tls.key
	ClientCertSecretRef *TLSClientCertRef `json:"clientCertSecretRef,omitempty"`
}

type TLSClientCertRef struct {
	Name string `json:"name"`

	// +kubebuilder:default="tls.crt"
	CertKey string `json:"certKey,omitempty"`

	// +kubebuilder:default="tls.key"
	KeyKey string `json:"keyKey,omitempty"`
}

type TemplateSpec struct {
	Template string `json:"template"`
}

type ValueFrom struct {
	SecretKeyRef *SecretKeyRef `json:"secretKeyRef,omitempty"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ResourceActionStatus struct {
	Executions []ExecutionRecord  `json:"executions,omitempty"`
	LastError  string             `json:"lastError,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type ResourceAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceActionSpec   `json:"spec,omitempty"`
	Status ResourceActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceActionList contains a list of ResourceAction.
type ResourceActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceAction `json:"items"`
}

type ExecutionRecord struct {
	ResourceUID string      `json:"resourceUID"`
	Event       string      `json:"event"`
	ExecutedAt  metav1.Time `json:"executedAt"`
}

func init() {
	SchemeBuilder.Register(&ResourceAction{}, &ResourceActionList{})
}
