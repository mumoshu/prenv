package ghactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/google/go-github/v56/github"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/envvar"
)

const (
	// EventTypeApply is the event type of the GitHub Actions repository_dispatch event
	// sent by prenv.
	EventTypeApply   = "prenv-apply"
	EventTypeDestroy = "prenv-destroy"
)

// SendRepositoryDispatch sends a GitHub Actions repository_dispatch event to the target repository.
// The clientPayload is a JSON-encoded Inputs, which contains the raw_config field,
// which is parsed by UnmarshalClientPayload in the target repository.
func SendRepositoryDispatch(ctx context.Context, eventType string, d config.RepositoryDispatch, in Inputs) error {
	token := os.Getenv(envvar.GitHubToken)
	return sendRepositoryDispatch(ctx, d.Owner, d.Repo, token, eventType, in)
}

// sendRepositoryDispatch sends a GitHub Actions repository_dispatch event to the target repository.
// The event contains the given clientPayload.
// clientPayload is usually a JSON-encoded string created from Inputs, which contains the raw_config field,
// which UnmarshalClientPayload reads to get the configuration sent from the source repository.
func sendRepositoryDispatch(ctx context.Context, owner, repo, token, eventType string, clientPayload interface{}) error {
	if token == "" {
		return fmt.Errorf("missing required GitHub token for sending repository_dispatch to %s/%s", owner, repo)
	}

	client := config.NewGitHubClient()

	payload, err := json.Marshal(clientPayload)
	if err != nil {
		return err
	}

	raw := json.RawMessage(payload)

	if _, _, err := client.Repositories.Dispatch(ctx, owner, repo, github.DispatchRequestOptions{
		EventType:     eventType,
		ClientPayload: &raw,
	}); err != nil && !errors.Is(err, &github.AcceptedError{}) {
		return err
	}

	return nil
}
