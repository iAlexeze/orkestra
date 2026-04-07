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
Used for Helm chart tracking and upgrades.
*/}}
{{- define "orkestra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
Used for resource identification and grouping.
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
MUST remain stable across upgrades (do not change).
*/}}
{{- define "orkestra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "orkestra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
─────────────────────────────────────────────────────────────────────────────
RUNTIME HELPER FUNCTIONS
─────────────────────────────────────────────────────────────────────────────
*/}}

{{/*
Runtime ServiceAccount name.
Returns the name of the ServiceAccount for the Orkestra runtime.
*/}}
{{- define "orkestra.runtimeServiceAccountName" -}}
{{- if .Values.runtime.serviceAccount.name }}
{{- .Values.runtime.serviceAccount.name }}
{{- else }}
{{- printf "%s-runtime" (include "orkestra.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Runtime image — respects tag override, falls back to appVersion.
Returns the full container image URL for the Orkestra runtime.
*/}}
{{- define "orkestra.runtimeImage" -}}
{{- if .Values.runtime.image.tag }}
{{- printf "%s:%s" .Values.runtime.image.repository .Values.runtime.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.runtime.image.repository .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Leader election namespace — defaults to release namespace.
Returns where the leader election Lease should be stored.
*/}}
{{- define "orkestra.leaderElectionNamespace" -}}
{{- default .Release.Namespace .Values.runtime.leaderElection.namespace }}
{{- end }}

{{/*
Katalog ConfigMap name.
Returns the name of the ConfigMap containing the Katalog definition.
*/}}
{{- define "orkestra.katalogConfigMapName" -}}
{{- if .Values.runtime.katalog.existingConfigMap }}
{{- .Values.runtime.katalog.existingConfigMap }}
{{- else }}
{{- printf "%s-katalog" (include "orkestra.fullname" .) }}
{{- end }}
{{- end }}

{{/*
─────────────────────────────────────────────────────────────────────────────
CONTROL CENTER HELPER FUNCTIONS
─────────────────────────────────────────────────────────────────────────────
*/}}

{{/*
Control Center ServiceAccount name.
Returns the name of the ServiceAccount for the Control Center.
Note: Control Center typically needs minimal to no RBAC permissions.
*/}}
{{- define "orkestra.ccServiceAccountName" -}}
{{- if .Values.controlCenter.serviceAccount.name }}
{{- .Values.controlCenter.serviceAccount.name }}
{{- else }}
{{- printf "%s-cc" (include "orkestra.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Control Center image — respects tag override, falls back to appVersion.
Returns the full container image URL for the Orkestra Control Center.
*/}}
{{/*
Control Center image — respects tag override, falls back to appVersion.
*/}}
{{- define "orkestra.ccImage" -}}
{{- if .Values.controlCenter.image.tag }}
{{- printf "%s:%s" .Values.controlCenter.image.repository .Values.controlCenter.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.controlCenter.image.repository .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Control Center URL list.
Returns a comma-separated string of Orkestra runtime URLs to monitor.
Supports both single URL (orkestraURL) and list (orkestraURLs) formats.
*/}}
{{- define "orkestra.ccURLs" -}}
{{- if .Values.controlCenter.config.orkestraURLs }}
{{- $urls := .Values.controlCenter.config.orkestraURLs }}
{{- $result := "" }}
{{- range $i, $url := $urls }}
{{- if $i }}{{ $result = printf "%s,%s" $result $url }}{{ else }}{{ $result = $url }}{{ end }}
{{- end }}
{{- $result }}
{{- else if .Values.controlCenter.config.orkestraURL }}
{{- .Values.controlCenter.config.orkestraURL }}
{{- else }}
{{- printf "http://%s-runtime:8080" (include "orkestra.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Control Center port.
Returns the port number, with environment variable fallback.
*/}}
{{- define "orkestra.ccPort" -}}
{{- .Values.controlCenter.config.port | default 8090 }}
{{- end }}