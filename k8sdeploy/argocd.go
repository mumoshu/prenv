package k8sdeploy

import (
	"fmt"

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

type ArgoCDApp struct {
	config.ArgoCDApp

	// Name is the name of the ArgoCD application.
	// It will be NameBase-PullRequestNumber by default.
	Name string

	// The following fields are set by LoadEnvVars.

	// GitHubSHA is the SHA of the commit to be deployed.
	GitHubSHA string
	// PullRequestNumber is the number of the pull request to be deployed.
	PullRequestNumber int
}

func (a *ArgoCDApp) LoadEnvVars() error {
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

	return nil
}

func (a *ArgoCDApp) Validate() error {
	if err := a.ArgoCDApp.Validate(); err != nil {
		return err
	}

	if a.GitHubSHA == "" {
		return fmt.Errorf("githubSHA is required. Set GITHUB_SHA env var")
	}

	if a.PullRequestNumber == 0 {
		return fmt.Errorf("pullRequestNumber is required")
	}

	return nil
}
