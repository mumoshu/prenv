package config

import "fmt"

type Environment struct {
	// BaseName is the base name of the Per-Pull Request Environment.
	// This is used to generate the name of the Per-Pull Request Environment.
	// The generated environment name is then used to generate the name of the ArgoCD application.
	BaseName string `yaml:"name"`

	// ArgoCDApp is the ArgoCD application that deploys the Kubernetes applications.
	// You either need to specify the ArgoCDApp for each service or the only ArgoCDApp for the environment.
	// If you specify the ArgoCDApp for the environment, the ArgoCDApp for each service is ignored.
	// This is basically populated when your service is a monolith.
	ArgoCDApp *ArgoCDApp `yaml:"argocdApp"`

	// Services is a map of microservices that are deployed to the Per-Pull Request Environment.
	// You either need to specify the ArgoCDApp for each service or the only ArgoCDApp for the environment.
	// If you specify the ArgoCDApp for each service, the ArgoCDApp for the environment is ignored.
	// This is basically populated when your service is composed of multiple microservices.
	Services map[string]Service `yaml:"services"`
}

// Service is a microservice that is deployed to the Per-Pull Request Environment.
type Service struct {
	// ArgoCDApp is the ArgoCD application that deploys the Kubernetes applications
	// for this microservice.
	ArgoCDApp ArgoCDApp `yaml:"argocdApp"`
}

type ArgoCDApp struct {
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
