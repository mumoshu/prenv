package config

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	FlagSourceQueueURL       = "source-queue-url"
	FlagDestinationQueueURLs = "destination-queue-urls"
)

type SQSForwarder struct {
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

func (f *SQSForwarder) Validate() error {
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

func (f *SQSForwarder) BuildDeployConfig(defaults Deploy) (*Deploy, error) {
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
