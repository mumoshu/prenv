package config

const (
	DefaultImage = "mumoshu/prenv:latest"
)

// KubernetesResources represents the desired state of the Kubernetes resources
// to be a part of the infrastructure.
type KubernetesResources struct {
	// Delegate is the gitops config that is used to deploy the Kubernetes resources.
	//
	// If Delegate is not specified, the Kubernetes resources are deployed directly using
	// the built-in Kubernetes provisioner.
	//
	// If Delegate is specified, the Kubernetes resources are deployed using the gitops config,
	// which means that "this" prenv run (re)generates the Kubernetes manifest files that contain
	// inputs deducated from the environment and the configuration, and then commits and pushes.
	// It's the responsibility of the CD system of the target gitops repository to deploy the Kubernetes resources.
	Delegate *Delegate `yaml:"gitOps"`

	// Image is the docker image to be used for the Kubernetes applications.
	// It's supposed to be a prenv image.
	// Defaults to mumoshu/prenv:latest.
	Image           string                `yaml:"image"`
	SQSForwarder    SQSForwarder          `yaml:"sqsForwarder"`
	OutgoingWebhook OutgoingWebhookServer `yaml:"outgoingWebhook"`
}
