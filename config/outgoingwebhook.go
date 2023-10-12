package config

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
)

type OutgoingWebhookServer struct {
	// The URL of the Slack webhook.
	WebhookURL string `yaml:"webhookURL"`
	// The channel to send the message to.
	Channel string `yaml:"channel"`
	// The username to send the message as.
	Username string `yaml:"username"`
}

const (
	FlagWebhookURL = "webhook-url"
	FlagChannel    = "channel"
	FlagUsername   = "username"
)

func (s *OutgoingWebhookServer) BuildDeployConfig(defaults Deploy) (*Deploy, error) {
	if err := s.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	c := defaults.Clone()
	c.Name = "outgoing-webhook"
	c.Command = "prenv"
	c.Args = []string{
		"outgoing-webhook",
		"--" + FlagWebhookURL, s.WebhookURL,
		"--" + FlagChannel, s.Channel,
		"--" + FlagUsername, s.Username,
	}
	return &c, nil
}

func (o *OutgoingWebhookServer) String() string {
	return fmt.Sprintf("OutgoingWebhook{WebhookURL: %s, Channel: %s, Username: %s}", o.WebhookURL, o.Channel, o.Username)
}

func (o *OutgoingWebhookServer) Validate() error {
	if o.WebhookURL == "" {
		return errors.New("webhook_url is required")
	}

	if _, err := url.Parse(o.WebhookURL); err != nil {
		return errors.Wrap(err, "failed to parse webhook_url")
	}

	if o.Channel == "" {
		return errors.New("channel is required")
	}

	if o.Username == "" {
		return errors.New("username is required")
	}

	return nil
}
