package config

const (
	DefaultImage = "mumoshu/prenv:latest"
)

// KubernetesResources represents the desired state of the Kubernetes resources
// to be a part of the infrastructure.
type KubernetesResources struct {
	// Image is the docker image to be used for the Kubernetes applications.
	// It's supposed to be a prenv image.
	// Defaults to mumoshu/prenv:latest.
	Image           string                `yaml:"image"`
	SQSForwarder    SQSForwarder          `yaml:"sqsForwarder"`
	OutgoingWebhook OutgoingWebhookServer `yaml:"outgoingWebhook"`
}
