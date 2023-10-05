package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/goccy/go-yaml"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/env"
	"github.com/mumoshu/prenv/infra"
	"github.com/mumoshu/prenv/outgoingwebhook"
	"github.com/mumoshu/prenv/sqsforwarder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "prenv"}
	rootCmd.AddCommand(NewCmdInit())
	rootCmd.AddCommand(NewCmdDeinit())
	rootCmd.AddCommand(NewCmdApply())
	rootCmd.AddCommand(NewCmdDestroy())
	rootCmd.AddCommand(NewCmdSQSForwarder())
	rootCmd.AddCommand(NewCmdOutgoingWebhook())
	ctx := newSignalContext()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func newSignalContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	return ctx
}

func getConfig() (*config.Config, error) {
	var cfg config.Config

	f, err := os.Open("prenv.yaml")
	if err != nil {
		return nil, err
	}

	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode yaml: %w", err)
	}

	return &cfg, nil
}

func runE(fn func(context.Context) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd.Context()); err != nil {
			logrus.Error(err)
			return err
		}

		return nil
	}
}

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize prenv",
		Long:  "ensures that the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` are deployed to your Kubernetes cluster.",
		RunE: runE(func(ctx context.Context) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			return infra.Reconcile(ctx, *cfg)
		}),
	}

	return cmd
}

func NewCmdDeinit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deinit",
		Short: "Deinitialize prenv",
		Long:  "deletes the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` from your Kubernetes cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			return infra.Deinit(*cfg)
		},
	}
	return cmd
}

func NewCmdApply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply prenv",
		Long:  "deploys your application to the Per-Pull Request Environment.",
		Run: func(cmd *cobra.Command, args []string) {
			env.Apply()
		},
	}

	return cmd
}

func NewCmdDestroy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy prenv",
		Long:  "undeploys your application from the Per-Pull Request Environment.",
		Run: func(cmd *cobra.Command, args []string) {
			env.Destroy()
		},
	}

	return cmd
}

func NewCmdSQSForwarder() *cobra.Command {
	sf := &sqsforwarder.Forwarder{}

	cmd := &cobra.Command{
		Use:   "sqs-forwarder",
		Short: "SQS Forwarder",
		Long:  "starts a daemon that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sf.Run(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&sf.SourceQueueURL, sqsforwarder.FlagSourceQueueURL, "", "The URL of the source SQS queue.")
	cmd.Flags().StringSliceVar(&sf.DestinationQueueURLs, sqsforwarder.FlagDestinationQueueURLs, []string{}, "The URLs of the destination SQS queues.")
	cmd.Flags().Int64Var(&sf.MaxNumberOfMessages, "max-number-of-messages", 1, "The maximum number of messages to receive from the source queue.")
	cmd.Flags().Int64Var(&sf.VisibilityTimeoutSeconds, "visibility-timeout-seconds", 30, "The duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.")
	cmd.Flags().Int64Var(&sf.WaitTimeSeconds, "wait-time-seconds", 20, "The duration (in seconds) for which the call waits for a message to arrive in the queue before returning.")
	cmd.Flags().Int64Var(&sf.SleepSeconds, "sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after receiving a message from the source queue.")
	cmd.Flags().Int64Var(&sf.ReceiveMessageFailureSleepSeconds, "receive-message-failure-sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after failing to receive a message from the source queue. This is to prevent the daemon from spamming the source queue with ReceiveMessage requests.")
	cmd.Flags().Int64Var(&sf.SendMessageFailureSleepSeconds, "send-message-failure-sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after failing to send a message to a destination queue. This is to prevent the daemon from spamming the destination queue with SendMessage requests.")
	cmd.Flags().Int64Var(&sf.DeleteMessageFailureSleepSeconds, "delete-message-failure-sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after failing to delete a message from the source queue. This is to prevent the daemon from spamming the source queue with DeleteMessage requests.")
	cmd.Flags().StringSliceVar(&sf.MessageAttributeNames, "message-attribute-names", []string{}, "The message attribute names to receive from the source queue.")
	cmd.Flags().StringVar(&sf.AWSRegion, "aws-region", "", "The AWS region to use.")
	cmd.Flags().StringVar(&sf.AWSProfile, "aws-profile", "", "The AWS profile to use.")
	cmd.Flags().StringVar(&sf.LogLevel, "log-level", "info", "The log level to use. Valid values are \"debug\", \"info\", \"warn\", \"error\", and \"fatal\".")

	return cmd
}

func NewCmdOutgoingWebhook() *cobra.Command {
	owh := &outgoingwebhook.Server{}

	cmd := &cobra.Command{
		Use:   "outgoing-webhook",
		Short: "Outgoing Webhook",
		Long:  "starts a web server receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return owh.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&owh.WebhookURL, outgoingwebhook.FlagWebhookURL, "", "The URL of the Slack webhook.")
	cmd.Flags().StringVar(&owh.Channel, outgoingwebhook.FlagChannel, "", "The channel to send the message to.")
	cmd.Flags().StringVar(&owh.Username, outgoingwebhook.FlagUsername, "", "The username to send the message as.")

	return cmd
}
