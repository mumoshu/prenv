package k8sdeploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
)

const (
	tempDirPattern = "k8sdeploy"
)

type M struct {
	// Name is the name of the Kubernetes application.
	Name string

	// Template is the Go template to be used to render the Kubernetes manifests.
	Template string

	// TemplateData is the data to be used to render the Kubernetes manifests.
	TemplateData interface{}
}

// Manifests generates Kubernetes manifests for the given Kubernetes application.
// The manifests are written to a temporary directory.
// It's the caller's responsibility to delete the directory.
// This package provides a helper function Cleanup for that purpose.
func Manifests(ms ...M) (*string, error) {
	d, err := CreateTempDir()
	if err != nil {
		return nil, err
	}

	dir := *d

	for _, m := range ms {
		files, err := generateManifests(m.Name, m.Template, m.TemplateData)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			if err := os.WriteFile(filepath.Join(dir, f.Path), []byte(f.Content), 0644); err != nil {
				return nil, err
			}
		}
	}

	return &dir, nil
}

func Cleanup(dir *string) error {
	if dir == nil {
		return nil
	}

	if *dir == "" {
		return nil
	}

	if !strings.Contains(*dir, tempDirPattern) {
		return nil
	}

	return os.RemoveAll(*dir)
}

type file struct {
	Path    string
	Content string
}

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

func generateManifests(name, deployTemplate string, c interface{}) ([]file, error) {
	if name == "" {
		return nil, fmt.Errorf("name must not be empty: config=%v", c)
	}

	yamlFile := name + ".yaml"
	m := template.New(yamlFile)
	m = m.Funcs(template.FuncMap{
		"b64enc": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"split": func(sep, s string) []string {
			return strings.Split(s, sep)
		},
	})
	m, err := m.Parse(deployTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err := m.Execute(&buf, c); err != nil {
		return nil, err
	}

	return []file{
		{
			Path:    yamlFile,
			Content: buf.String(),
		},
	}, nil
}

// Apply deploys the application by generating manifests and kubectl-applying them to the Kubernetes cluster.
func Apply(ctx context.Context, ms ...M) error {
	manifestsDir, err := Manifests(ms...)
	if err != nil {
		return fmt.Errorf("unable to generate Kubernetes manifests: %w", err)
	}

	defer func() {
		if err := Cleanup(manifestsDir); err != nil {
			logrus.Error(err)
		}
	}()

	if err := KubectlApply(ctx, *manifestsDir); err != nil {
		return fmt.Errorf("unable to apply Kubernetes manifests: %w", err)
	}

	return nil
}

// Delete deletes the application by generating manifests and kubectl-deleting them from the Kubernetes cluster.
func Delete(ctx context.Context, ms ...M) error {
	manifestsDir, err := Manifests(ms...)
	if err != nil {
		return fmt.Errorf("unable to generate Kubernetes manifests: %w", err)
	}

	defer func() {
		if err := Cleanup(manifestsDir); err != nil {
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
