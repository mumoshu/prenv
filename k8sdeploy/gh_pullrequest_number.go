package k8sdeploy

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const (
	EnvVarGitHubEventPath = "GITHUB_EVENT_PATH"
)

// See https://github.com/actions/checkout/issues/58#issuecomment-589447479
func GetPullRequestNumber() (*int, error) {
	data, err := ioutil.ReadFile(os.Getenv(EnvVarGitHubEventPath))
	if err != nil {
		return nil, err
	}

	var event map[string]interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	if event["pull_request"] == nil {
		return nil, nil
	}

	pr := event["pull_request"].(map[string]interface{})
	number := int(pr["number"].(float64))
	return &number, nil
}
