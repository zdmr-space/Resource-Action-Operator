{{- define "ra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ra.fullname" -}}
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

{{- define "ra.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "ra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "ra.resourceActionName" -}}
{{- default (include "ra.fullname" .) .Values.resourceAction.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ra.serviceAccountName" -}}
{{- if .Values.job.serviceAccount.create -}}
{{- default (printf "%s-runner" (include "ra.fullname" .)) .Values.job.serviceAccount.name -}}
{{- else -}}
{{- .Values.job.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "ra.jobImage" -}}
{{- $registry := .Values.job.image.registry | trimSuffix "/" -}}
{{- $repository := .Values.job.image.repository -}}
{{- $image := $repository -}}
{{- if $registry -}}
{{- $image = printf "%s/%s" $registry $repository -}}
{{- end -}}
{{- if .Values.job.image.digest -}}
{{- printf "%s@%s" $image .Values.job.image.digest -}}
{{- else -}}
{{- printf "%s:%s" $image .Values.job.image.tag -}}
{{- end -}}
{{- end -}}
