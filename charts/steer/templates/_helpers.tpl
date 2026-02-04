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
Chart name and version.
*/}}
{{- define "steer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "steer.labels" -}}
helm.sh/chart: {{ include "steer.chart" . }}
{{ include "steer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
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
Service account name
*/}}
{{- define "steer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "steer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Compute manager --metrics-bind-address
*/}}
{{- define "steer.metricsBindAddress" -}}
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
{{- define "steer.metricsServicePort" -}}
{{- if .Values.metrics.service.port }}
{{- .Values.metrics.service.port -}}
{{- else }}
{{- .Values.metrics.port -}}
{{- end }}
{{- end }}
