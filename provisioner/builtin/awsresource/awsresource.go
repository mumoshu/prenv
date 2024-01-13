package awsresource

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/mumoshu/prenv/awsclicompat"
)

// AWSResources manages lifecycles of AWS resources
// used by prenv.
// It currently manages AWS SQS queues.
type AWSResources struct {
	AWSRegion  string
	AWSProfile string

	once sync.Once
	svc  *sqs.SQS
}

// EnsureQueueCreated ensures that an SQS queue with the given name or URL exists.
// If the queue does not exist and the name of the queue and create=true are specified, it will create the queue.
// If the queue does not exist and the URL of the queue is specified, it will return an error.
// If the queue exists, it will return the URL of the queue.
// It returns the URL of the queue on success.
func (r *AWSResources) EnsureQueueCreated(ctx context.Context, nameOrURL string, create bool) (string, error) {
	svc, err := r.createOrGetService()
	if err != nil {
		return "", err
	}

	if nameOrURL == "" {
		return "", fmt.Errorf("nameOrURL must be specified")
	}

	isURL := strings.HasPrefix(nameOrURL, "https://")

	if isURL {
		_, err := svc.GetQueueAttributesWithContext(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(nameOrURL),
			AttributeNames: []*string{
				aws.String("QueueArn"),
			},
		})

		if err != nil {
			return "", err
		}

		return nameOrURL, nil
	}

	res, err := svc.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(nameOrURL),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
				if !create {
					return "", fmt.Errorf("queue %s does not exist. Specify create=true if you want to create it", nameOrURL)
				}

				res, err := svc.CreateQueueWithContext(ctx, &sqs.CreateQueueInput{
					QueueName: aws.String(nameOrURL),
				})

				if err != nil {
					return "", err
				}

				return *res.QueueUrl, nil
			}
		}

		return "", err
	}

	return *res.QueueUrl, nil
}

// EnsureQueueDeleted ensures that an SQS queue with the given name or URL does not exist.
// If the queue exists, it will be deleted.
func (r *AWSResources) EnsureQueueDeleted(nameOrURL string) error {
	svc, err := r.createOrGetService()
	if err != nil {
		return err
	}

	isURL := strings.HasPrefix(nameOrURL, "https://")

	var url string

	if !isURL {
		res, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
			QueueName: aws.String(nameOrURL),
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
					return nil
				}
			}

			return err
		}

		url = *res.QueueUrl
	}

	_, err = svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(url),
	})

	if err != nil {
		return err
	}

	return nil
}

// createOrGetService creates or gets the SQS service.
func (r *AWSResources) createOrGetService() (*sqs.SQS, error) {
	var err error

	r.once.Do(func() {
		sess := awsclicompat.NewSession(r.AWSRegion, r.AWSProfile, "")

		r.svc = sqs.New(sess, &aws.Config{
			Region: aws.String(r.AWSRegion),
		})
	})

	if err != nil {
		return nil, fmt.Errorf("unable to create or get service: %w", err)
	}

	return r.svc, err
}
