package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mumoshu/prenv/envvar"
)

type Event struct {
	// Action is usually "workflow_dispatch" or "repository_dispatch"
	// in cases Event is concerned with.
	Action string `json:"action"`

	// Inputs for workflow_dispatch
	Inputs json.RawMessage `json:"inputs"`

	// ClientPayload for repository_dispatch
	ClientPayload json.RawMessage `json:"client_payload"`

	PullRequest map[string]interface{} `json:"pull_request"`
}

func GetEventPayload() (map[string]interface{}, error) {
	path := os.Getenv(envvar.GitHubEventPath)
	if path == "" {
		return nil, fmt.Errorf("%s must not be empty", envvar.GitHubEventPath)
	}

	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", envvar.GitHubEventPath, err)
	}

	var payload = map[string]interface{}{}

	if err := json.Unmarshal(f, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", envvar.GitHubEventPath, err)
	}

	return payload, nil
}
