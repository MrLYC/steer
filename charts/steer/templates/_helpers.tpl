{{/*
展开 chart 名称。
*/}}
{{- define "steer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
创建默认的 fully qualified 应用名称。
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
通用标签。
*/}}
{{- define "steer.labels" -}}
helm.sh/chart: {{ include "steer.chart" . }}
{{ include "steer.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector 标签。
*/}}
{{- define "steer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "steer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
创建用于 chart label 的 chart 名称与版本。
*/}}
{{- define "steer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
存放历史 ConfigMap 的命名空间。
默认使用 Release 命名空间。
*/}}
{{- define "steer.historyNamespace" -}}
{{- if .Values.history.namespace }}
{{- .Values.history.namespace }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
命名空间选择器的标签 key 与 value。
*/}}
{{- define "steer.namespaceSelectorKey" -}}
{{- .Values.namespaceSelector.key }}
{{- end }}

{{- define "steer.namespaceSelectorValue" -}}
{{- .Values.namespaceSelector.value }}
{{- end }}
