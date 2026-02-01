{{/*
Expand the name of the chart.
*/}}
{{- define "steer-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "steer-operator.fullname" -}}
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
Chart name and version.
*/}}
{{- define "steer-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "steer-operator.labels" -}}
helm.sh/chart: {{ include "steer-operator.chart" . }}
{{ include "steer-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "steer-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "steer-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name
*/}}
{{- define "steer-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "steer-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Compute manager --metrics-bind-address
*/}}
{{- define "steer-operator.metricsBindAddress" -}}
{{- if .Values.metrics.bindAddress }}
{{- .Values.metrics.bindAddress -}}
{{- else if .Values.metrics.listenOnAllInterfaces }}
{{- printf "0.0.0.0:%d" (int .Values.metrics.port) -}}
{{- else }}
{{- printf "127.0.0.1:%d" (int .Values.metrics.port) -}}
{{- end }}
{{- end }}

{{/*
Compute metrics Service port
*/}}
{{- define "steer-operator.metricsServicePort" -}}
{{- if .Values.metrics.service.port }}
{{- .Values.metrics.service.port -}}
{{- else }}
{{- .Values.metrics.port -}}
{{- end }}
{{- end }}
