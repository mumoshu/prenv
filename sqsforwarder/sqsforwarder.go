package sqsforwarder

import (
	"context"
	"strings"
	"time"

	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/mumoshu/prenv/awsclicompat"
	"github.com/mumoshu/prenv/k8sdeploy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Forwarder is a daemon that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
// The SQS queue to forward messages from is specified by the SourceQueueURL field.
// The downstream, Per-Pull Request Environments' SQS queues are specified by the DestinationQueueURLs field.
type Forwarder struct {
	// The URL of the SQS queue to forward messages from.
	SourceQueueURL string `yaml:"sourceQueueURL"`
	// The URLs of the downstream, Per-Pull Request Environments' SQS queues.
	DestinationQueueURLs []string `yaml:"desinationQueueURLs"`
	// The maximum number of messages to receive from the source queue at a time.
	MaxNumberOfMessages int64 `yaml:"maxNumberOfMessages"`
	// The duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.
	VisibilityTimeoutSeconds int64 `yaml:"visibilityTimeout"`
	// The duration (in seconds) for which the call waits for a message to arrive in the queue before returning.
	WaitTimeSeconds int64 `yaml:"waitTimeSeconds"`
	// The duration (in seconds) that the daemon sleeps after receiving a message from the source queue.
	SleepSeconds int64 `yaml:"sleepSeconds"`
	// The duration (in seconds) that the daemon sleeps after failing to receive a message from the source queue.
	// This is to prevent the daemon from spamming the source queue with ReceiveMessage requests.
	ReceiveMessageFailureSleepSeconds int64 `yaml:"receiveMessageFailureSleepSeconds"`
	// The duration (in seconds) that the daemon sleeps after failing to send a message to a destination queue.
	// This is to prevent the daemon from spamming the destination queue with SendMessage requests.
	SendMessageFailureSleepSeconds int64 `yaml:"sendMessageFailureSleepSeconds"`
	// The duration (in seconds) that the daemon sleeps after failing to delete a message from the source queue.
	// This is to prevent the daemon from spamming the source queue with DeleteMessage requests.
	DeleteMessageFailureSleepSeconds int64 `yaml:"deleteMessageFailureSleepSeconds"`
	// The message attribute names to receive from the source queue.
	MessageAttributeNames []string `yaml:"messageAttributeNames"`
	// The AWS region to use.
	AWSRegion string `yaml:"awsRegion"`
	// The AWS profile to use.
	AWSProfile string `yaml:"awsProfile"`
	// The log level to use.
	// Valid values are "debug", "info", "warn", "error", and "fatal".
	LogLevel string `yaml:"logLevel"`
}

const (
	FlagSourceQueueURL       = "source-queue-url"
	FlagDestinationQueueURLs = "destination-queue-urls"
)

func (f *Forwarder) BuildDeployConfig(defaults k8sdeploy.Config) (*k8sdeploy.Config, error) {
	if err := f.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	c := defaults.Clone()
	c.Name = "sqs-forwarder"
	c.Command = "prenv"
	c.Args = []string{
		"sqs-forwarder",
		"--" + FlagSourceQueueURL, f.SourceQueueURL,
		"--" + FlagDestinationQueueURLs, strings.Join(f.DestinationQueueURLs, ","),
	}
	return &c, nil
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

func (f *Forwarder) Validate() error {
	if f.SourceQueueURL == "" {
		return errors.New("source queue URL is required")
	}

	if len(f.DestinationQueueURLs) == 0 {
		return errors.New("at least one destination queue URL is required")
	}

	if f.MaxNumberOfMessages <= 0 {
		return errors.New("max number of messages must be greater than 0")
	}

	if f.VisibilityTimeoutSeconds <= 0 {
		return errors.New("visibility timeout must be greater than 0")
	}

	if f.WaitTimeSeconds <= 0 {
		return errors.New("wait time must be greater than 0")
	}

	if f.SleepSeconds <= 0 {
		return errors.New("sleep must be greater than 0")
	}

	if f.ReceiveMessageFailureSleepSeconds <= 0 {
		return errors.New("receive message failure sleep must be greater than 0")
	}

	if f.SendMessageFailureSleepSeconds <= 0 {
		return errors.New("send message failure sleep must be greater than 0")
	}

	if f.DeleteMessageFailureSleepSeconds <= 0 {
		return errors.New("delete message failure sleep must be greater than 0")
	}

	return nil
}
