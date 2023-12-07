package state

import (
	"context"
	"os"
)

const (
	EnvVarPrefix = "PRENV_"

	EnvVarConfigMapName        = EnvVarPrefix + "CONFIGMAP_NAME"
	EnvVarGitRepoURL           = EnvVarPrefix + "GIT_REPO_URL"
	EnvVarBaseBranch           = EnvVarPrefix + "BASE_BRANCH"
	EnvVarGitRoot              = EnvVarPrefix + "GIT_ROOT"
	EnvVarCommitAuthorUserName = EnvVarPrefix + "COMMIT_AUTHOR_USER_NAME"

	EnvVarGitHubToken = "GITHUB_TOKEN"

	// EnvVarStateFilePath is the path to the file that stores the state of the environment.
	//
	// This file is usually stored in either a local git repository or a remote git repository.
	//
	// When EnvVarGitRepoURL is set, this file is stored in the remote git repository.
	// When EnvVarGitRepoURL is not set, this file is stored in the local git repository.
	//
	// If the file is stored in the local git repository, prenv internally deduce the remote repository
	// URL from the local git repository URL, and push the local git repository to the remote repository.
	EnvVarStateFilePath = EnvVarPrefix + "STATE_FILE_PATH"
)

// NewStore returns a Store implementation based on the environment variables.
// If EnvVarConfigMapName is set, it returns a ConfigMapStore.
// If EnvVarGitRepoURL is set, it returns a GitStore.
// Otherwise, it returns a YAMLFileStore.
func NewStore() Store {
	if cmName := os.Getenv(EnvVarConfigMapName); cmName != "" {
		return &ConfigMapStore{
			Name: cmName,
		}
	}

	gitRepoURL := os.Getenv(EnvVarGitRepoURL)
	if gitRepoURL == "" {
		actionsGitHubRepo := os.Getenv("GITHUB_REPOSITORY")
		if actionsGitHubRepo != "" {
			gitRepoURL = "https://github.com/" + actionsGitHubRepo
		}
	}

	if gitRepoURL != "" {
		return newGitStore(
			os.Getenv(EnvVarCommitAuthorUserName),
			os.Getenv(EnvVarGitHubToken),
			gitRepoURL,
			os.Getenv(EnvVarBaseBranch),
			os.Getenv(EnvVarStateFilePath),
			os.Getenv(EnvVarGitRoot),
		)
	}

	return &YAMLFileStore{
		Path: os.Getenv(EnvVarStateFilePath),
	}
}

type Store interface {
	AddEnvironmentName(ctx context.Context, name string) error
	DeleteEnvironmentName(ctx context.Context, name string) error
	ListEnvironmentNames(ctx context.Context) ([]string, error)
}

type datastore interface {
	load(context.Context, []byte) (*State, error)
	getState(context.Context) (*State, error)
	setState(context.Context, *State) error
	getData() []byte
}
