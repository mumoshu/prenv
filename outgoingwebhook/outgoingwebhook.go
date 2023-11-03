package outgoingwebhook

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/mumoshu/prenv/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Server is a webhook server that sends a Slack message to a channel
// when the webhook is triggered.
//
// The webhook expects a POST request with any form values.
// Each form value becomes a Slack attachment field.
type Server struct {
	*config.OutgoingWebhookServer
}

func NewOutgoingWebhook(webhookURL, channel, username string) *Server {
	return &Server{
		&config.OutgoingWebhookServer{
			WebhookURL: webhookURL,
			Channel:    channel,
			Username:   username,
		},
	}
}

// Run starts a webhook server that sends a Slack message to a channel when the
// webhook is triggered.
//
// The webhook expects a POST request with any form values.
// Each form value becomes a Slack attachment field.
//
// The webhook server is started on the given address.
// The webhook server is stopped when the context is canceled.
//
// The webhook server returns an error if the configuration is invalid or if the
// webhook server fails to start.
//
// If the webhook URL is empty and the environment variable SLACK_WEBHOOK_URL is set,
// the webhook URL is set to the value of the environment variable.
func (o *Server) Run(ctx context.Context) error {
	if e := os.Getenv(config.EnvSlackWebhookURL); e != "" {
		if o.WebhookURL != "" {
			logrus.Warnf("%s is set but webhook-url is also set. Using webhook-url.", config.EnvSlackWebhookURL)
		} else {
			o.WebhookURL = e
		}
	}

	if err := o.Validate(); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	logrus.WithField("address", o.Address()).Info("starting outgoing webhook server")
	srv := http.Server{
		Addr:    o.Address(),
		Handler: o,
	}

	go func() {
		<-ctx.Done()
		logrus.Info("stopping outgoing webhook server")
		if err := srv.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("failed to stop outgoing webhook server")
		}
	}()

	return srv.ListenAndServe()
}

// Address returns the address the webhook server is listening on.
func (o *Server) Address() string {
	return ":8080"
}

// ServeHTTP implements http.Handler.
func (o *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := o.handleRequest(w, r); err != nil {
		logrus.WithError(err).Error("failed to handle request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (o *Server) handleRequest(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return errors.Wrap(err, "failed to parse form")
	}

	var fields []map[string]string
	for k, v := range r.Form {
		fields = append(fields, map[string]string{
			"title": k,
			"value": strings.Join(v, ", "),
			"short": "true",
		})
	}

	attachment := map[string]interface{}{
		"fallback": "New message",
		"color":    "#36a64f",
		"fields":   fields,
	}

	attachments := []map[string]interface{}{attachment}

	message := map[string]interface{}{
		"channel":     o.Channel,
		"username":    o.Username,
		"text":        "New message",
		"attachments": attachments,
	}

	b, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	resp, err := http.Post(o.WebhookURL, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return errors.Wrap(err, "failed to send message to Slack")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to send message to Slack: %s", resp.Status)
	}

	return nil
}
