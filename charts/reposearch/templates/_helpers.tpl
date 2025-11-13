{{/*
Expand the name of the chart.
*/}}
{{- define "reposearch.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "reposearch.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "reposearch.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "reposearch.labels" -}}
helm.sh/chart: {{ include "reposearch.chart" . }}
{{ include "reposearch.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "reposearch.selectorLabels" -}}
app.kubernetes.io/name: {{ include "reposearch.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "reposearch.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (printf "%s-indexer" (include "reposearch.fullname" .)) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Common template aliases for Bitnami compatibility
*/}}
{{- define "common.names.fullname" -}}
{{- include "reposearch.fullname" . }}
{{- end }}

{{- define "common.names.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}

{{- define "common.labels.standard" -}}
{{- include "reposearch.labels" .context }}
{{- end }}

{{- define "reposearch.imagePullSecrets" -}}
{{- end }}

{{- define "reposearch.indexer.image" -}}
{{- printf "%s/%s:%s" .Values.indexer.image.registry .Values.indexer.image.repository .Values.indexer.image.tag }}
{{- end }}

{{- define "common.tplvalues.render" -}}
{{- end }}

{{- define "common.tplvalues.merge" -}}
{{- end }}

{{- define "common.affinities.pods" -}}
{{- end }}

{{- define "common.affinities.nodes" -}}
{{- end }}

{{- define "common.compatibility.renderSecurityContext" -}}
{{- end }}

{{- define "reposearch.defaultInitContainers.volumePermissions" -}}
{{- end }}

{{- define "common.resources.preset" -}}
{{- end }}
