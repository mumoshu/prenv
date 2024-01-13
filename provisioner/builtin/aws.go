package builtin

import (
	"context"
	"fmt"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/builtin/awsresource"
	"github.com/mumoshu/prenv/provisioner/plugin"
)

type BuiltinAWSProvisioner struct {
	Config config.AWSResources
}

func (p *BuiltinAWSProvisioner) Render(ctx context.Context, dir string) (*plugin.RenderResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *BuiltinAWSProvisioner) Apply(ctx context.Context, _ *plugin.RenderResult) (*plugin.Result, error) {
	res, err := ensureAWSResourcesCreated(ctx, p.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure resources to be created: %w", err)
	}

	// Print the URLs of the SQS queues to stdout.
	// The URLs are used by the sqs-forwarder.
	fmt.Printf("SQS_SOURCE_QUEUE_URL=%s\n", res.SQSSourceQueueURL)
	fmt.Printf("SQS_DESTINATION_QUEUE_URL=%s\n", res.SQSDestinationQueueURL)
	for i, url := range res.SQSDestinationQueueURLs {
		fmt.Printf("SQS_DESTINATION_QUEUE_URL_%d=%s\n", i, url)
	}

	var r plugin.Result

	r.Outputs = map[string]plugin.Output{}

	r.Outputs["sqsDestinationQueueURL"] = plugin.Output{Type: "sqsQueue", Value: res.SQSDestinationQueueURL}
	r.Outputs["sqsDestinationQueueURLs"] = plugin.Output{Type: "[]sqsQueue", Value: res.SQSDestinationQueueURLs}
	r.Outputs["sqsSourceQueueURL"] = plugin.Output{Type: "sqsQueue", Value: res.SQSSourceQueueURL}

	return &r, nil
}

type AWSResources struct {
	SQSSourceQueueURL       string
	SQSDestinationQueueURL  string
	SQSDestinationQueueURLs []string
}

func ensureAWSResourcesCreated(ctx context.Context, c config.AWSResources) (*AWSResources, error) {
	resources := awsresource.AWSResources{
		AWSRegion: "ap-northeast-1",
	}

	sourceQueueURL, err := resources.EnsureQueueCreated(ctx, c.SourceQueueURL, c.SourceQueueCreate)
	if err != nil {
		return nil, fmt.Errorf("unable to get or create source queue: %w", err)
	}

	destinationQueueURL, err := resources.EnsureQueueCreated(ctx, c.DestinationQueueURL, c.DestinationQueueCreate)
	if err != nil {
		return nil, fmt.Errorf("unable to get or create destination queue: %w", err)
	}

	var destinationQueueURLs []string

	for _, url := range c.DestinationQueueNames {
		destinationQueueURL, err := resources.EnsureQueueCreated(ctx, url, c.DestinationQueuesCreate)
		if err != nil {
			return nil, fmt.Errorf("unable to create destination queue %q: %w", url, err)
		}
		destinationQueueURLs = append(destinationQueueURLs, destinationQueueURL)
	}

	return &AWSResources{
		SQSSourceQueueURL:       sourceQueueURL,
		SQSDestinationQueueURL:  destinationQueueURL,
		SQSDestinationQueueURLs: destinationQueueURLs,
	}, nil
}

func (p *BuiltinAWSProvisioner) Destroy(ctx context.Context) (*plugin.Result, error) {
	awsResourceOptions := p.Config

	resources := awsresource.AWSResources{
		AWSRegion: "ap-northeast-1",
	}

	if awsResourceOptions.SourceQueueDelete {
		if err := resources.EnsureQueueDeleted(awsResourceOptions.SourceQueueURL); err != nil {
			return nil, err
		}
	}

	if awsResourceOptions.DestinationQueueDelete {
		if err := resources.EnsureQueueDeleted(awsResourceOptions.DestinationQueueURL); err != nil {
			return nil, err
		}
	}

	for _, name := range awsResourceOptions.DestinationQueueNames {
		if awsResourceOptions.DestinationQueueDelete {
			if err := resources.EnsureQueueDeleted(name); err != nil {
				return nil, err
			}
		}
	}

	return &plugin.Result{}, nil
}
