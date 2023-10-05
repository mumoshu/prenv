package k8sdeploy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
)

// Config represents the desired state of the Kubernetes application.
// Each Kubernetes application is a set of Kubernetes resources.
type Config struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Replicas  *int     `yaml:"replicas"`
	Command   string   `yaml:"command"`
	Image     string   `yaml:"image"`
	Args      []string `yaml:"args"`
	Port      *int     `yaml:"port"`
}

func (c *Config) Clone() Config {
	clone := *c
	if c.Replicas != nil {
		replicas := *c.Replicas
		clone.Replicas = &replicas
	}
	clone.Args = append([]string{}, c.Args...)
	return clone
}

const (
	tempDirPattern = "k8sdeploy"
)

// Manifests generates Kubernetes manifests for the given Kubernetes application.
// The manifests are written to a temporary directory.
// It's the caller's responsibility to delete the directory.
// This package provides a helper function Cleanup for that purpose.
func Manifests(configs ...*Config) (*string, error) {
	f, err := os.CreateTemp("", tempDirPattern)
	if err != nil {
		return nil, err
	}

	dir := f.Name()

	if err := os.Remove(dir); err != nil {
		return nil, fmt.Errorf("unable to replace the temp file %s with a directory: %w", dir, err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	for _, c := range configs {
		files, err := generateManifests(dir, c)
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

func generateManifests(dir string, c *Config) ([]file, error) {
	const (
		deployTemplate = `apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Namespace }}
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
        args:
        {{- range $arg := .Args }}
        - {{ $arg }}
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
	yamlFile := c.Name + ".yaml"
	m := template.New(yamlFile)
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
