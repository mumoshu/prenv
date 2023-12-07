package config

// AWSResources represents the desired state of the AWS resources
// to be a part of the infrastructure.
type AWSResources struct {
	// GitOps is the gitops config that is used to deploy the AWS resources.
	//
	// If GitOps is not specified, the AWS resources are deployed directly using either
	// Terraform or the built-in AWS provisioner.
	//
	// If GitOps is specified, the AWS resources are deployed using the gitops config,
	// which means that "this" prenv run (re)generates the tfvars file that contains
	// inputs deducated from the environment and the configuration, and then commits and pushes.
	// It's the responsibility of the CD system of the target gitops repository to deploy the AWS resources.
	GitOps *GitOps `yaml:"gitOps"`

	// If true, the source queue is created.
	// If false, the SourceQueueURL must be specified, the queue needs to exist, and is used as the source queue.
	// In case you want to use an existing queue, you can specify the URL of the queue as the SourceQueueURL.
	SourceQueueCreate bool `yaml:"sourceQueueCreate"`
	// SourceQueueDelete specifies whether the source queue is deleted when the infrastructure is deinitialized.
	// Do not set this to true if you want to use an existing queue as the source queue,
	// or if you want to keep the source queue after the infrastructure is deinitialized.
	SourceQueueDelete bool `yaml:"sourceQueueDelete"`
	// In case you want to use an existing queue, you can specify the URL of the queue as the SourceQueueURL.
	// The URL must be in the format of https://sqs.ap-northeast-1.amazonaws.com/123456789012/queue-name.
	// The queue must be in the same region as the AWSRegion.
	// The queue must be in the same AWS account as the AWSProfile.
	//
	// You can also specify the name of the queue as the SourceQueueURL.
	// In this case, the queue is created in the AWS account specified by the AWSProfile.
	SourceQueueURL string `yaml:"sourceQueueURL"`

	// If true, the destination queue is created.
	// If false, the DestinationQueueURL must be specified, the queue needs to exist, and is used as the destination queue.
	DestinationQueueCreate bool `yaml:"destinationQueueCreate"`
	// DestinationQueueDelete specifies whether the destination queue is deleted when the infrastructure is deinitialized.
	// Do not set this to true if you want to use an existing queue as the destination queue,
	// or if you want to keep the destination queue after the infrastructure is deinitialized.
	DestinationQueueDelete bool `yaml:"destinationQueueDelete"`
	// In case you want to use an existing queue, you can specify the URL of the queue as the DestinationQueueURL.
	// The URL must be in the format of https://sqs.ap-northeast-1.amazonaws.com/123456789012/queue-name.
	// The queue must be in the same region as the AWSRegion.
	// The queue must be in the same AWS account as the AWSProfile.
	//
	// You can also specify the name of the queue as the DestinationQueueURL.
	// In this case, the queue is created in the AWS account specified by the AWSProfile.
	DestinationQueueURL string `yaml:"destinationQueueURL"`

	// If true, the destination queues are created.
	// If false, the DestinationQueueURLs must be specified, the queues need to exist, and are used as the destination queues.
	DestinationQueuesCreate bool     `yaml:"destinationQueuesCreate"`
	DestinationQueueNames   []string `yaml:"destinationQueueURLs"`
}
