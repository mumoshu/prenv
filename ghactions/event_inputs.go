package ghactions

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/envvar"
)

func UnmarshalInputs(dest interface{}) error {
	event, err := unmarshalEvent()
	if err != nil {
		return err
	}

	if len(event.Inputs) == 0 {
		return fmt.Errorf("no workflow_dispatch inputs found in %s", envvar.GitHubEventPath)
	}

	if err := json.Unmarshal(event.Inputs, dest); err != nil {
		return err
	}

	return nil
}

func UnmarshalClientPayload(dest interface{}) error {
	event, err := unmarshalEvent()
	if err != nil {
		return err
	}

	if len(event.ClientPayload) == 0 {
		return fmt.Errorf("no repository_dispatch client_payload found in %s", envvar.GitHubEventPath)
	}

	if err := json.Unmarshal(event.ClientPayload, dest); err != nil {
		return err
	}

	return nil
}

func GetAction() (string, error) {
	event, err := unmarshalEvent()
	if err != nil {
		return "", err
	}

	return event.Action, nil
}

func unmarshalEvent() (*config.Event, error) {
	ghEventPath := os.Getenv(envvar.GitHubEventPath)
	if ghEventPath == "" {
		return nil, fmt.Errorf("env var %s is not set", envvar.GitHubEventPath)
	}

	data, err := os.ReadFile(ghEventPath)
	if err != nil {
		return nil, err
	}

	var event config.Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %s: %w", ghEventPath, err)
	}

	return &event, nil
}
