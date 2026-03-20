{{/*
Expand the name of the chart.
*/}}
{{- define "orkestra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "orkestra.fullname" -}}
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
Chart label — name + version.
*/}}
{{- define "orkestra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "orkestra.labels" -}}
helm.sh/chart: {{ include "orkestra.chart" . }}
{{ include "orkestra.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — used in Deployment selector and Service selector.
Must remain stable across upgrades.
*/}}
{{- define "orkestra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "orkestra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "orkestra.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "orkestra.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Image — respects tag override, falls back to appVersion.
*/}}
{{- define "orkestra.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Leader election namespace — defaults to release namespace.
*/}}
{{- define "orkestra.leaderElectionNamespace" -}}
{{- default .Release.Namespace .Values.leaderElection.namespace }}
{{- end }}

{{/*
ConfigMap name for the Katalog.
*/}}
{{- define "orkestra.katalogConfigMapName" -}}
{{- if .Values.katalog.existingConfigMap }}
{{- .Values.katalog.existingConfigMap }}
{{- else }}
{{- printf "%s-katalog" (include "orkestra.fullname" .) }}
{{- end }}
{{- end }}
