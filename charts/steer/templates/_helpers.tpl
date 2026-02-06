{{/*
Expand the name of the chart.
*/}}
{{- define "steer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "steer.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "steer.labels" -}}
helm.sh/chart: {{ include "steer.chart" . }}
{{ include "steer.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "steer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "steer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "steer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
The namespace where history ConfigMaps are stored.
Defaults to the release namespace.
*/}}
{{- define "steer.historyNamespace" -}}
{{- if .Values.history.namespace }}
{{- .Values.history.namespace }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Namespace selector label key and value.
*/}}
{{- define "steer.namespaceSelectorKey" -}}
{{- .Values.namespaceSelector.key }}
{{- end }}

{{- define "steer.namespaceSelectorValue" -}}
{{- .Values.namespaceSelector.value }}
{{- end }}
