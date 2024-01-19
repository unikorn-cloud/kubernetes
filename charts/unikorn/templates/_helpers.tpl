{{/*
Expand the name of the chart.
*/}}
{{- define "unikorn.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "unikorn.fullname" -}}
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
{{- define "unikorn.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "unikorn.labels" -}}
helm.sh/chart: {{ include "unikorn.chart" . }}
{{ include "unikorn.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "unikorn.selectorLabels" -}}
app.kubernetes.io/name: {{ include "unikorn.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "unikorn.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "unikorn.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the container images
*/}}
{{- define "unikorn.defaultRepositoryPath" -}}
{{- if .Values.repository }}
{{- printf "%s/%s" .Values.repository .Values.organization }}
{{- else }}
{{- .Values.organization }}
{{- end }}
{{- end }}

{{- define "unikorn.projectManagerImage" -}}
{{- .Values.projectManager.image | default (printf "%s/unikorn-project-manager:%s" (include "unikorn.defaultRepositoryPath" .) (.Values.tag | default .Chart.Version)) }}
{{- end }}

{{- define "unikorn.controlPlaneManagerImage" -}}
{{- .Values.controlPlaneManager.image | default (printf "%s/unikorn-control-plane-manager:%s" (include "unikorn.defaultRepositoryPath" .) (.Values.tag | default .Chart.Version)) }}
{{- end }}

{{- define "unikorn.clusterManagerImage" -}}
{{- .Values.clusterManager.image | default (printf "%s/unikorn-cluster-manager:%s" (include "unikorn.defaultRepositoryPath" .) (.Values.tag | default .Chart.Version)) }}
{{- end }}

{{- define "unikorn.monitorImage" -}}
{{- .Values.monitor.image | default (printf "%s/unikorn-monitor:%s" (include "unikorn.defaultRepositoryPath" .) (.Values.tag | default .Chart.Version)) }}
{{- end }}

{{- define "unikorn.serverImage" -}}
{{- .Values.server.image | default (printf "%s/unikorn-server:%s" (include "unikorn.defaultRepositoryPath" .) (.Values.tag | default .Chart.Version)) }}
{{- end }}

{{/*
Create Prometheus labels
*/}}
{{- define "unikorn.prometheusServiceSelector" -}}
prometheus.spjmurray.co.uk/app: unikorn
{{- end }}

{{- define "unikorn.prometheusJobLabel" -}}
prometheus.spjmurray.co.uk/job
{{- end }}

{{- define "unikorn.prometheusLabels" -}}
{{ include "unikorn.prometheusServiceSelector" . }}
{{ include "unikorn.prometheusJobLabel" . }}: {{ .job }}
{{- end }}

{{/*
Create image pull secrets
*/}}
{{- define "unikorn.imagePullSecrets" -}}
{{- if .Values.imagePullSecret -}}
- name: {{ .Values.imagePullSecret }}
{{ end }}
{{- if .Values.dockerConfig -}}
- name: docker-config
{{- end }}
{{- end }}
