package state

import (
	"context"
	"os"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/envvar"
)

// NewStore returns a Store implementation based on the environment variables.
// If EnvVarConfigMapName is set, it returns a ConfigMapStore.
// If EnvVarGitRepoURL is set, it returns a GitStore.
// Otherwise, it returns a YAMLFileStore.
func NewStore(_ config.Config) Store {
	if cmName := os.Getenv(envvar.ConfigMapName); cmName != "" {
		return &ConfigMapStore{
			Name: cmName,
		}
	}

	gitRepoURL := os.Getenv(envvar.GitRepoURL)
	if gitRepoURL == "" {
		actionsGitHubRepo := os.Getenv("GITHUB_REPOSITORY")
		if actionsGitHubRepo != "" {
			gitRepoURL = "https://github.com/" + actionsGitHubRepo
		}
	}

	// if gitRepoURL != "" {
	// 	return newGitStore(
	// 		os.Getenv(envvar.GitCommitAuthorUserName),
	// 		os.Getenv(envvar.GitHubToken),
	// 		gitRepoURL,
	// 		os.Getenv(envvar.BaseBranch),
	// 		os.Getenv(envvar.StateFilePath),
	// 		os.Getenv(envvar.GitRoot),
	// 	)
	// }

	path := os.Getenv(envvar.StateFilePath)
	if path == "" {
		path = "prenv.state.yaml"
	}

	return &YAMLFileStore{
		Path: path,
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
