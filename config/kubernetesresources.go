package config

import (
	"github.com/mumoshu/prenv/outgoingwebhook"
	"github.com/mumoshu/prenv/sqsforwarder"
)

const (
	DefaultImage = "mumoshu/prenv:latest"
)

// KubernetesResources represents the desired state of the Kubernetes resources
// to be a part of the infrastructure.
type KubernetesResources struct {
	// Image is the docker image to be used for the Kubernetes applications.
	// It's supposed to be a prenv image.
	// Defaults to mumoshu/prenv:latest.
	Image           string                 `yaml:"image"`
	SQSForwarder    sqsforwarder.Forwarder `yaml:"sqsForwarder"`
	OutgoingWebhook outgoingwebhook.Server `yaml:"outgoingWebhook"`
}
