package config

import "fmt"

type Component struct {
	// NamePrefix is the base name of the Per-Pull Request Environment.
	// This is used to generate the name of the Per-Pull Request Environment.
	// The generated environment name is then used to generate the name of the ArgoCD application.
	NamePrefix string `yaml:"namePrefix,omitempty"`

	// AWSResources is the configuration for the AWS resources that are used by prenv.
	// This includes the SQS queues that are used by the sqs-forwarder and by
	// the pull-request environments.
	AWSResources *AWSResources `yaml:"awsResources,omitempty"`

	// KubernetesResources is the configuration for the Kubernetes resources that are used by prenv.
	// This includes the Kubernetes resources that are used by the sqs-forwarder and by
	// outgoing-webhook, but not the pull-request environments.
	KubernetesResources *KubernetesResources `yaml:"kubernetesResources,omitempty"`

	Render *Render `yaml:"render,omitempty"`

	ArgoCD `yaml:"argocd,omitempty"`

	// Components is a map of microservices that are deployed to the Per-Pull Request Environment.
	// You either need to specify the ArgoCDApp for each service or the only ArgoCDApp for the environment.
	// If you specify the ArgoCDApp for each service, the ArgoCDApp for the environment is ignored.
	// This is basically populated when your service is composed of multiple microservices.
	Components map[string]Component `yaml:"components,omitempty"`
}

type Render struct {
	Delegate `yaml:",inline"`

	Files []RenderedFile `yaml:"files,omitempty"`
}

type RenderedFile struct {
	Name            string `yaml:"name,omitempty"`
	NameTemplate    string `yaml:"nameTemplate,omitempty"`
	ContentTemplate string `yaml:"contentTemplate"`
}

// ArgoCD is a set of configuration and apps for ArgoCD.
type ArgoCD struct {
	// GitOps is the configuration for the gitops config that is used to deploy the environment.
	GitOps *Delegate `yaml:"gitOps"`

	// App is the ArgoCD application that deploys the Kubernetes applications.
	// You either need to specify the App for each service or the only App for the environment.
	// If you specify the App for the environment, the App for each service is ignored.
	// This is basically populated when your service is a monolith.
	App *ArgoCDApp `yaml:"app"`
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
