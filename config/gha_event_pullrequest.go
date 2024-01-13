package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mumoshu/prenv/envvar"
)

// See https://github.com/actions/checkout/issues/58#issuecomment-589447479
func GetPullRequestNumber() (*int, error) {
	ghEventPath := os.Getenv(envvar.GitHubEventPath)
	if ghEventPath == "" {
		return nil, fmt.Errorf("env var %s is not set", envvar.GitHubEventPath)
	}

	data, err := os.ReadFile(ghEventPath)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	if event.PullRequest == nil {
		return nil, nil
	}

	number := int(event.PullRequest["number"].(float64))
	return &number, nil
}
