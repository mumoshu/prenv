package config

import "fmt"

type Environment struct {
	// NameTemplate is the Go template used to generate the name of the environment
	NameTemplate string
	// ArgoCDApp is the ArgoCD application that deploys the Kubernetes applications
	ArgoCDApp ArgoCDApp `yaml:"argocdApp"`
}

type ArgoCDApp struct {
	// Name is the base name of the Kubernetes application.
	// The actual name of the Kubernetes application will be
	// Name-PullRequestNumber by default.
	Name string `yaml:"name"`
	// Namespace is the namespace of the ArgoCD application.
	Namespace string `yaml:"namespace"`
	// DestinationNamespace is the namespace of the Kubernetes application that is deployed by ArgoCD.
	DestinationNamespace string `yaml:"destinationNamespace"`
	// DestinationServer is the URL of the Kubernetes cluster that is deployed by ArgoCD.
	DestinationServer string `yaml:"destinationServer"`
	// Path is the path to the directory that contains the Kubernetes manifests.
	Path string `yaml:"path"`
	// RepoURL is the URL of the git repository that contains the Kubernetes manifests.
	RepoURL string `yaml:"repoURL"`
	// TargetRevision is the revision of the git repository that contains the Kubernetes manifests.
	TargetRevision string `yaml:"targetRevision"`
	// Image is the docker image to be used for the Kubernetes applications.
	Image string `yaml:"image"`
}

func (a *ArgoCDApp) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("name is required")
	}

	if a.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if a.DestinationNamespace == "" {
		return fmt.Errorf("destinationNamespace is required")
	}

	if a.DestinationServer == "" {
		return fmt.Errorf("destinationServer is required")
	}

	if a.Path == "" {
		return fmt.Errorf("path is required")
	}

	if a.RepoURL == "" {
		return fmt.Errorf("repoURL is required")
	}

	if a.TargetRevision == "" {
		return fmt.Errorf("targetRevision is required")
	}

	if a.Image == "" {
		return fmt.Errorf("image is required")
	}

	return nil
}
