package k8sdeploy

import (
	"github.com/mumoshu/prenv/config"
)

const TemplateArgoCDApp = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  destination:
    namespace: {{ .DestinationNamespace }}-{{ .PullRequest.Number }}
    server: {{ .DestinationServer }}
  project: default
  source:
    repoURL: {{ .RepoURL }}
    targetRevision: {{ .TargetRevision }}
    path: {{ .Path }}
    kustomize:
      # https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/#setting-the-manifests-namespace
      namespace: {{ .DestinationNamespace }}-{{ .PullRequest.Number }}
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

	Environment config.EnvArgs
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
