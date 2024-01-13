package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/mumoshu/prenv/apps/outgoingwebhook"
	"github.com/mumoshu/prenv/apps/sqsforwarder"
	"github.com/mumoshu/prenv/build"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Main() error {
	var rootCmd = &cobra.Command{
		Use:     "prenv",
		Version: build.Version(),
	}
	rootCmd.AddCommand(NewCmdApply())
	rootCmd.AddCommand(NewCmdDestroy())
	rootCmd.AddCommand(NewCmdAction())
	rootCmd.AddCommand(NewCmdSQSForwarder())
	rootCmd.AddCommand(NewCmdOutgoingWebhook())
	ctx := newSignalContext()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return err
	}
	return nil
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

func runE(fn func(context.Context) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd.Context()); err != nil {
			logrus.Error(err)
			return err
		}

		return nil
	}
}

func NewCmdApply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply prenv",
		Long:  "deploys your application to the Per-Pull Request Environment.",
		RunE: runE(func(ctx context.Context) error {
			cfg, err := provisioner.ChainFromEnv()
			if err != nil {
				return err
			}

			return cfg.Apply(ctx)
		}),
	}

	return cmd
}

func NewCmdDestroy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy prenv",
		Long:  "undeploys your application from the Per-Pull Request Environment.",
		RunE: runE(func(ctx context.Context) error {
			cfg, err := provisioner.ChainFromEnv()
			if err != nil {
				return err
			}

			return cfg.Destroy(ctx)
		}),
	}

	return cmd
}

func NewCmdAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: "prenv gh action",
		Long:  "Runs either apply or destroy depending on the event type of the GitHub Actions repository_dispatch event sent by prenv.",
		RunE: runE(func(ctx context.Context) error {
			cfg, err := provisioner.ChainFromEnv()
			if err != nil {
				return err
			}

			return cfg.Action(ctx)
		}),
	}

	return cmd
}

func NewCmdSQSForwarder() *cobra.Command {
	var c config.SQSForwarder

	cmd := &cobra.Command{
		Use:   "sqs-forwarder",
		Short: "SQS Forwarder",
		Long:  "starts a daemon that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.",
		RunE: func(cmd *cobra.Command, args []string) error {
			sf := sqsforwarder.Forwarder{
				SQSForwarder: &c,
			}
			return sf.Run(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&c.SourceQueueURL, config.FlagSourceQueueURL, "", "The URL of the source SQS queue.")
	cmd.Flags().StringSliceVar(&c.DestinationQueueURLs, config.FlagDestinationQueueURLs, []string{}, "The URLs of the destination SQS queues.")
	cmd.Flags().Int64Var(&c.MaxNumberOfMessages, "max-number-of-messages", 1, "The maximum number of messages to receive from the source queue.")
	cmd.Flags().Int64Var(&c.VisibilityTimeoutSeconds, "visibility-timeout-seconds", 30, "The duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.")
	cmd.Flags().Int64Var(&c.WaitTimeSeconds, "wait-time-seconds", 20, "The duration (in seconds) for which the call waits for a message to arrive in the queue before returning.")
	cmd.Flags().Int64Var(&c.SleepSeconds, "sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after receiving a message from the source queue.")
	cmd.Flags().Int64Var(&c.ReceiveMessageFailureSleepSeconds, "receive-message-failure-sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after failing to receive a message from the source queue. This is to prevent the daemon from spamming the source queue with ReceiveMessage requests.")
	cmd.Flags().Int64Var(&c.SendMessageFailureSleepSeconds, "send-message-failure-sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after failing to send a message to a destination queue. This is to prevent the daemon from spamming the destination queue with SendMessage requests.")
	cmd.Flags().Int64Var(&c.DeleteMessageFailureSleepSeconds, "delete-message-failure-sleep-seconds", 10, "The duration (in seconds) that the daemon sleeps after failing to delete a message from the source queue. This is to prevent the daemon from spamming the source queue with DeleteMessage requests.")
	cmd.Flags().StringSliceVar(&c.MessageAttributeNames, "message-attribute-names", []string{}, "The message attribute names to receive from the source queue.")
	cmd.Flags().StringVar(&c.AWSRegion, config.FlagAWSRegion, "", "The AWS region to use.")
	cmd.Flags().StringVar(&c.AWSProfile, "aws-profile", "", "The AWS profile to use.")
	cmd.Flags().StringVar(&c.LogLevel, "log-level", "info", "The log level to use. Valid values are \"debug\", \"info\", \"warn\", \"error\", and \"fatal\".")

	return cmd
}

func NewCmdOutgoingWebhook() *cobra.Command {
	var c config.OutgoingWebhookServer

	cmd := &cobra.Command{
		Use:   "outgoing-webhook",
		Short: "Outgoing Webhook",
		Long:  "starts a web server receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.",
		RunE: func(cmd *cobra.Command, args []string) error {
			owh := &outgoingwebhook.Server{
				OutgoingWebhookServer: &c,
			}
			return owh.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&c.WebhookURL, config.FlagWebhookURL, "", "The URL of the Slack webhook.")
	cmd.Flags().StringVar(&c.Channel, config.FlagChannel, "", "The channel to send the message to.")
	cmd.Flags().StringVar(&c.Username, config.FlagUsername, "", "The username to send the message as.")

	return cmd
}
