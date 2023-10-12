package sqsforwarder

import (
	"context"
	"strings"
	"time"

	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/mumoshu/prenv/awsclicompat"
	"github.com/mumoshu/prenv/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Forwarder is a daemon that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
// The SQS queue to forward messages from is specified by the SourceQueueURL field.
// The downstream, Per-Pull Request Environments' SQS queues are specified by the DestinationQueueURLs field.
type Forwarder struct {
	*config.SQSForwarder
}

// Run runs a daemon that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
// This is a blocking call.
// The daemon stops when the context is canceled.
//
// The daemon returns an error if the configuration is invalid or if the daemon fails to start.
func (f *Forwarder) Run(ctx context.Context) error {
	sess := awsclicompat.NewSession(f.AWSRegion, f.AWSProfile, "")

	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	level, err := logrus.ParseLevel(f.LogLevel)
	if err != nil {
		return errors.Wrap(err, "invalid log level")
	}
	log.SetLevel(level)

	s := sqs.New(sess)

	if err := f.Validate(); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	for {
		input := &sqs.ReceiveMessageInput{
			QueueUrl:            &f.SourceQueueURL,
			MaxNumberOfMessages: &f.MaxNumberOfMessages,
			VisibilityTimeout:   &f.VisibilityTimeoutSeconds,
			WaitTimeSeconds:     &f.WaitTimeSeconds,
		}
		if len(f.MessageAttributeNames) > 0 {
			for _, messageAttributeName := range f.MessageAttributeNames {
				input.MessageAttributeNames = append(input.MessageAttributeNames, &messageAttributeName)
			}
		}
		messages, err := s.ReceiveMessageWithContext(ctx, input)
		if err != nil {
			if strings.Contains(err.Error(), "AWS.SimpleQueueService.NonExistentQueue") {
				return errors.Wrap(err, "source queue does not exist")
			}
			if strings.Contains(err.Error(), "RequestCanceled") {
				log.WithError(err).Info("context canceled")
				return nil
			}
			// Log the error and retry later.
			log.WithError(err).Error("failed to receive message from source queue")
			time.Sleep(time.Duration(f.ReceiveMessageFailureSleepSeconds) * time.Second)
			continue
		}

		for _, message := range messages.Messages {
			for _, destinationQueueURL := range f.DestinationQueueURLs {
				_, err := s.SendMessageWithContext(ctx, &sqs.SendMessageInput{
					QueueUrl:    &destinationQueueURL,
					MessageBody: message.Body,
				})

				if err != nil {
					// Log the error and retry later.
					log.WithError(err).Error("failed to send message to destination queue")
					time.Sleep(time.Duration(f.SendMessageFailureSleepSeconds) * time.Second)
					continue
				}
			}

			_, err := s.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &f.SourceQueueURL,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				// Log the error and retry later.
				log.WithError(err).Error("failed to delete message from source queue")
				time.Sleep(time.Duration(f.DeleteMessageFailureSleepSeconds) * time.Second)
				continue
			}
		}

		t := time.NewTimer(time.Duration(f.SleepSeconds) * time.Second)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			continue
		}
	}
}
