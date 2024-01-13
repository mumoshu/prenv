package config

import "os"

const (
	EnvVarGitHubSHA = "GITHUB_SHA"
)

func GetSHA() (string, error) {
	return os.Getenv(EnvVarGitHubSHA), nil
}
