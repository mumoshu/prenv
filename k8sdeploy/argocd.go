package k8sdeploy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mumoshu/prenv/config"
)

const TemplateArgoCDApp = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  destination:
    namespace: {{ .DestinationNamespace }}-{{ .PullRequestNumber }}
    server: {{ .DestinationServer }}
  project: default
  source:
    repoURL: {{ .RepoURL }}
    targetRevision: {{ .TargetRevision }}
    path: {{ .Path }}
    kustomize:
      # https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/#setting-the-manifests-namespace
      namespace: {{ .DestinationNamespace }}-{{ .PullRequestNumber }}
      images:
       - '{{ .Image }}:{{ .GitHubSHA }}'
  syncPolicy:
    automated: {}
    syncOptions:
    - CreateNamespace=true
`

// AppParams is the parameters for the Kubernetes application to be deployed per pull request.
type AppParams struct {
	// Name is the name of the application.
	// This is used:
	// - For generating the name of the ArgoCD application.
	// - For generating the file name of the Kubernetes manifests.
	Name string

	// ShortName is the short name of the Kubernetes application.
	// It is used to generate Name from EnvParams.AppNameTemplate.
	ShortName string

	config.ArgoCDApp
	Environment EnvParams
}

func (a *AppParams) Validate() error {
	if err := a.ArgoCDApp.Validate(); err != nil {
		return err
	}

	if err := a.Environment.Validate(); err != nil {
		return err
	}

	return nil
}

// EnvParams is the parameters for the environment to be deployed per pull request.
type EnvParams struct {
	// Name is the name of the environment.
	// It will be NameBase-PullRequestNumber by default.
	Name string

	// AppNameTemplate is the Go template used to generate the name of the ArgoCD application.
	// It is `{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}` or `{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}-{{ .ShortName }} by default,
	AppNameTemplate string

	// The following fields are set by LoadEnvVars.

	// GitHubSHA is the SHA of the commit to be deployed.
	GitHubSHA string
	// PullRequestNumber is the number of the pull request to be deployed.
	PullRequestNumber int

	// GitHubActionsEventPayload is the payload of the GitHub Actions event.
	GitHubActionsEventPayload map[string]interface{}
}

// LoadEnvVarsAndEvent loads the environment variables and the GitHub Actions event payload.
// The loaded values are set to the EnvParams and therefore avaiable for Go templates used
// for generating the Kubernetes manifests.
func (a *EnvParams) LoadEnvVarsAndEvent() error {
	prNumber, err := GetPullRequestNumber()
	if err != nil {
		return err
	}

	sha, err := GetSHA()
	if err != nil {
		return err
	}

	a.PullRequestNumber = *prNumber
	a.GitHubSHA = sha

	p, err := GetGitHubActionsEventPayload()
	if err != nil {
		return err
	}

	a.GitHubActionsEventPayload = p

	return nil
}

func GetGitHubActionsEventPayload() (map[string]interface{}, error) {
	const (
		// https://docs.github.com/en/actions/reference/environment-variables#default-environment-variables
		githubEventPath = "GITHUB_EVENT_PATH"
	)

	path := os.Getenv(githubEventPath)
	if path == "" {
		return nil, fmt.Errorf("%s must not be empty", githubEventPath)
	}

	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", githubEventPath, err)
	}

	var payload = map[string]interface{}{}

	if err := json.Unmarshal(f, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", githubEventPath, err)
	}

	return payload, nil
}

func (a *EnvParams) Validate() error {
	if a.GitHubSHA == "" {
		return fmt.Errorf("githubSHA is required. Set GITHUB_SHA env var")
	}

	if a.PullRequestNumber == 0 {
		return fmt.Errorf("pullRequestNumber is required")
	}

	return nil
}
