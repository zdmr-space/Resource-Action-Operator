{{- define "rao.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "rao.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "rao.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "rao.labels" -}}
helm.sh/chart: {{ include "rao.chart" . }}
app.kubernetes.io/name: {{ include "rao.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "rao.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rao.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end -}}

{{- define "rao.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "rao.fullnameWithSuffix" (dict "context" . "suffix" "controller-manager")) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "rao.fullnameWithSuffix" -}}
{{- printf "%s-%s" (include "rao.fullname" .context) .suffix | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "rao.image" -}}
{{- $registry := .Values.image.registry | trimSuffix "/" -}}
{{- $repository := .Values.image.repository -}}
{{- $image := $repository -}}
{{- if $registry -}}
{{- $image = printf "%s/%s" $registry $repository -}}
{{- end -}}
{{- if .Values.image.digest -}}
{{- printf "%s@%s" $image .Values.image.digest -}}
{{- else -}}
{{- printf "%s:%s" $image .Values.image.tag -}}
{{- end -}}
{{- end -}}
