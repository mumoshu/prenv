package k8sdeploy

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mumoshu/prenv/render"
	"github.com/sirupsen/logrus"
)

const (
	TemplateDeployment = `apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Namespace }}
{{- if .SecretEnv }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
type: Opaque
data:
{{- range $key, $value := .SecretEnv }}
  {{ $key }}: {{ $value | b64enc }}
{{- end }}
{{- end }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  {{ if .Replicas -}}
  replicas: {{ .Replicas}}
  {{ end -}}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      containers:
      - name: {{ .Name }}
        image: {{ .Image }}
		{{- if .Port }}
        ports:
        - containerPort: {{ .Port }}
		{{- end }}
        command:
        - {{ .Command }}
        {{- if .Args }}
        args:
        {{- range $arg := .Args }}
        - {{ $arg }}
        {{- end }}
        {{- end }}
        {{- if or .Env .SecretEnv }}
        env:
        {{- range $key, $value := .Env }}
        - name: {{ $key }}
          value: {{ $value }}
        {{- end }}
        {{- range $key, $value := .SecretEnv }}
        - name: {{ $key }}
          valueFrom:
            secretKeyRef:
              name: {{ $.Name }}
              key: {{ $key }}
        {{- end }}
        {{- end }}
{{ if .Port -}}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  selector:
    app: {{ .Name }}
  ports:
  - protocol: TCP
    port: {{ .Port }}
    targetPort: {{ .Port }}
{{- end }}
`
)

// Apply deploys the application by generating manifests and kubectl-applying them to the Kubernetes cluster.
func Apply(ctx context.Context, ms ...render.Template) error {
	manifestsDir, err := render.ToTempDir(ms...)
	if err != nil {
		return fmt.Errorf("unable to generate Kubernetes manifests: %w", err)
	}

	defer func() {
		if err := render.Cleanup(manifestsDir); err != nil {
			logrus.Error(err)
		}
	}()

	if err := KubectlApply(ctx, *manifestsDir); err != nil {
		return fmt.Errorf("unable to apply Kubernetes manifests: %w", err)
	}

	return nil
}

// Delete deletes the application by generating manifests and kubectl-deleting them from the Kubernetes cluster.
func Delete(ctx context.Context, ms ...render.Template) error {
	manifestsDir, err := render.ToTempDir(ms...)
	if err != nil {
		return fmt.Errorf("unable to generate Kubernetes manifests: %w", err)
	}

	defer func() {
		if err := render.Cleanup(manifestsDir); err != nil {
			logrus.Error(err)
		}
	}()

	if err := KubectlDelete(ctx, *manifestsDir); err != nil {
		return fmt.Errorf("unable to delete Kubernetes resources: %w", err)
	}

	return nil
}

func KubectlApply(ctx context.Context, path string) error {
	k := &kubectl{}

	if err := k.Apply(ctx, path); err != nil {
		if strings.Contains(err.Error(), "error validating data") {
			var (
				causeFile                string
				errorValidatingFileRegex = regexp.MustCompile(`error validating "([^"]+)"`)
			)

			matches := errorValidatingFileRegex.FindStringSubmatch(err.Error())
			if len(matches) > 1 {
				causeFile = matches[1]
			}

			if causeFile != "" {
				logrus.Info("The following file failed validation:")
				logrus.Info(causeFile)
				logrus.Info("Content:")
				content, readFileErr := os.ReadFile(causeFile)
				if readFileErr != nil {
					logrus.Error(readFileErr)
					return err
				}
				logrus.Info(string(content))
				logrus.Info("This is likely due to a missing or invalid field in the file.")
				logrus.Info("Please check the file, fix the config file or file a bug report.")
			}
		}
		return err
	}

	return nil
}

func KubectlDelete(ctx context.Context, path string) error {
	k := &kubectl{}

	if err := k.Delete(ctx, path); err != nil {
		return err
	}

	return nil
}
