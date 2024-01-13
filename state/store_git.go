package state

import (
	"context"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/mumoshu/prenv/store"
)

type GitStore struct {
	// stateFilePath is the path to the file that contains the state of the gitops config
	stateFilePath string

	ds *store.Git
}

var _ Store = &GitStore{}

func newGitStore(commitAuthorUserName, githubToken, repo, baseBranch, stateFilePath, gitRoot string) *GitStore {
	var ds store.Git

	ds.Auth = &http.BasicAuth{
		Username: "prenvbot", // This can be anything except an empty string
		Password: githubToken,
	}
	ds.GitRepoURL = repo
	ds.AuthorName = commitAuthorUserName

	if gitRoot == "" {
		gitRoot = ".prenv/repos"
	}
	ds.GitRoot = gitRoot

	baseRefName := plumbing.Master
	if baseBranch != "" {
		baseRefName = plumbing.ReferenceName(baseBranch)
	}
	ds.BaseRefName = baseRefName

	var s GitStore

	s.stateFilePath = stateFilePath
	s.ds = &ds

	return &s
}

func (s *GitStore) AddEnvironmentName(ctx context.Context, name string) error {
	return s.ds.ModifyFile("add-env-"+name, s.stateFilePath, "Delete environment name "+name, func(data []byte) ([]byte, error) {
		ds := &yamlDataStore{}
		s, err := ds.load(context.Background(), data)
		if err != nil {
			return nil, err
		}

		s.AddEnvironmentName(name)

		if err := ds.setState(context.Background(), s); err != nil {
			return nil, err
		}

		return ds.getData(), nil
	})
}

func (s *GitStore) DeleteEnvironmentName(ctx context.Context, name string) error {
	return s.ds.ModifyFile("delete-env-"+name, s.stateFilePath, "Delete environment name "+name, func(data []byte) ([]byte, error) {
		ds := &yamlDataStore{}
		s, err := ds.load(context.Background(), data)
		if err != nil {
			return nil, err
		}

		s.DeleteEnvironmentName(name)

		if err := ds.setState(context.Background(), s); err != nil {
			return nil, err
		}

		return ds.getData(), nil
	})
}

func (s *GitStore) ListEnvironmentNames(ctx context.Context) ([]string, error) {
	state, err := s.getState(ctx)
	if err != nil {
		return nil, err
	}

	return state.EnvironmentNames, nil
}

func (s *GitStore) getState(ctx context.Context) (*State, error) {
	yamlData, err := s.ds.GetFileFromBranch("get-envs", s.stateFilePath)
	if err != nil {
		return nil, err
	}

	ds := &yamlDataStore{}
	return ds.load(ctx, yamlData)
}
