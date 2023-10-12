// Package "infra" provides the infrastructure for the application.
//
// There are two main functions: Init() and Deinit().
//
// Init() is called when the infrastructure is initialized.
// Deinit() is called when the infrastructure is deinitialized.
//
// The infrastructure is initialized before the firstapplication starts and deinitialized after all the applications are stopped.
// The infrastructure is initialized and deinitialized only once.
package infra

import (
	"context"
	"fmt"

	"github.com/mumoshu/prenv/awsresource"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/k8sdeploy"
	"github.com/mumoshu/prenv/state"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// Reconciles the infrastructure.
//
// It deploys the instructure, currently outgoingwebhook and sqs-forwarder,
// to the Kubernetes cluster.
// It also creates the necessary resources, currently the SQS queues, in the AWS account, if it is instructed to do so.
// The passed or created SQS queues are used by the sqs-forwarder.
// It returns an error if it fails to initialize the infrastructure.
//
// This function is supposed to be called before any pull-request env is created,
// and after each pull-request is opened.
// It is idempotent so you can call it multiple times, without fearing that it will create duplicate resources.
func Reconcile(ctx context.Context, cfg config.Config) error {
	store := &state.Store{}
	envNames, err := store.ListEnvironmentNames(ctx)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("unable to list enviroment names: %w", err)
		}
	}

	awsResourceOptions := cfg.AWSResources
	awsResourceOptions.DestinationQueueNames = envNames

	awsRes, err := deployAWSResources(ctx, awsResourceOptions)
	if err != nil {
		return fmt.Errorf("unable to deploy AWS resources: %w", err)
	}

	k8sRes := cfg.KubernetesResources
	if k8sRes.Image == "" {
		k8sRes.Image = config.DefaultImage
	}
	k8sRes.SQSForwarder.SourceQueueURL = awsRes.SQSSourceQueueURL
	k8sRes.SQSForwarder.DestinationQueueURLs = append(k8sRes.SQSForwarder.DestinationQueueURLs, awsRes.SQSDestinationQueueURL)
	k8sRes.SQSForwarder.DestinationQueueURLs = append(k8sRes.SQSForwarder.DestinationQueueURLs, awsRes.SQSDestinationQueueURLs...)

	if err := deployKubernetesResources(ctx, k8sRes); err != nil {
		return fmt.Errorf("unable to deploy Kubernetes resources: %w", err)
	}

	return nil
}

func deployKubernetesResources(ctx context.Context, k8sRes config.KubernetesResources) error {
	defaults := config.Deploy{
		Namespace: "prenv",
		Image:     k8sRes.Image,
	}

	sf, err := k8sRes.SQSForwarder.BuildDeployConfig(defaults)
	if err != nil {
		return fmt.Errorf("unable to build deploy config for sqs forwarder: %w", err)
	}
	ow, err := k8sRes.OutgoingWebhook.BuildDeployConfig(defaults)
	if err != nil {
		return fmt.Errorf("unable to build deploy config for outgoing webhook: %w", err)
	}

	if err := k8sdeploy.Apply(ctx,
		k8sdeploy.M{
			Name:         sf.Name,
			Template:     k8sdeploy.TemplateDeployment,
			TemplateData: sf,
		},
		k8sdeploy.M{
			Name:         ow.Name,
			Template:     k8sdeploy.TemplateDeployment,
			TemplateData: ow,
		},
	); err != nil {
		return fmt.Errorf("unable to apply Kubernetes manifests: %w", err)
	}

	return nil
}

func deployAWSResources(ctx context.Context, c config.AWSResources) (*AWSResources, error) {
	res, err := ensureAWSResourcesCreated(ctx, c)
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

	return res, nil
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

func Deinit(cfg config.Config) error {
	ctx := context.Background()
	store := &state.Store{}
	envNames, err := store.ListEnvironmentNames(ctx)
	if err != nil {
		return err
	}

	awsResourceOptions := cfg.AWSResources
	awsResourceOptions.DestinationQueueNames = envNames

	return destroy(awsResourceOptions)
}

func destroy(awsResourceOptions config.AWSResources) error {
	resources := awsresource.AWSResources{
		AWSRegion:  "ap-northeast-1",
		AWSProfile: "default",
	}

	if awsResourceOptions.SourceQueueDelete {
		if err := resources.EnsureQueueDeleted(awsResourceOptions.SourceQueueURL); err != nil {
			return err
		}
	}

	if awsResourceOptions.DestinationQueueDelete {
		if err := resources.EnsureQueueDeleted(awsResourceOptions.DestinationQueueURL); err != nil {
			return err
		}
	}

	for _, name := range awsResourceOptions.DestinationQueueNames {
		if awsResourceOptions.DestinationQueueDelete {
			if err := resources.EnsureQueueDeleted(name); err != nil {
				return err
			}
		}
	}

	return nil
}
