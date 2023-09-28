package main

import (
	"github.com/mumoshu/prenv/env"
	"github.com/mumoshu/prenv/infra"
	"github.com/mumoshu/prenv/outgoingwebhook"
	"github.com/mumoshu/prenv/sqsforwarder"
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
	rootCmd.Execute()
}

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize prenv",
		Long:  "ensures that the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` are deployed to your Kubernetes cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			infra.Init()
		},
	}

	return cmd
}

func NewCmdDeinit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deinit",
		Short: "Deinitialize prenv",
		Long:  "deletes the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` from your Kubernetes cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			infra.Deinit()
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
	cmd := &cobra.Command{
		Use:   "sqs-forwarder",
		Short: "SQS Forwarder",
		Long:  "starts a daemon that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sqsforwarder.Run()
		},
	}

	return cmd
}

func NewCmdOutgoingWebhook() *cobra.Command {
	owh := &outgoingwebhook.OutgoingWebhook{}

	cmd := &cobra.Command{
		Use:   "outgoing-webhook",
		Short: "Outgoing Webhook",
		Long:  "starts a web server receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return owh.Run()
		},
	}

	cmd.Flags().StringVar(&owh.WebhookURL, "webhook-url", "", "The URL of the Slack webhook.")
	cmd.Flags().StringVar(&owh.Channel, "channel", "", "The channel to send the message to.")
	cmd.Flags().StringVar(&owh.Username, "username", "", "The username to send the message as.")

	return cmd
}
