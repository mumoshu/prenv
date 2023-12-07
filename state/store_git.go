package state

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
)

type GitStore struct {
	// stateFilePath is the path to the file that contains the state of the gitops config
	stateFilePath string

	ds *gitDataStore
}

var _ Store = &GitStore{}

func newGitStore(commitAuthorUserName, githubToken, repo, baseBranch, stateFilePath, gitRoot string) *GitStore {
	var ds gitDataStore

	ds.auth = &http.BasicAuth{
		Username: "prenvbot", // This can be anything except an empty string
		Password: githubToken,
	}
	ds.gitRepoURL = repo
	ds.authorUserName = commitAuthorUserName
	ds.gitRoot = gitRoot

	baseRefName := plumbing.Master
	if baseBranch != "" {
		baseRefName = plumbing.ReferenceName(baseBranch)
	}
	ds.baseRefName = baseRefName

	var s GitStore

	s.stateFilePath = stateFilePath
	s.ds = &ds

	return &s
}

func (s *GitStore) AddEnvironmentName(ctx context.Context, name string) error {
	return s.ds.modifyState("add-env-"+name, s.stateFilePath, "Delete environment name "+name, func(data []byte) ([]byte, error) {
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
	return s.ds.modifyState("delete-env-"+name, s.stateFilePath, "Delete environment name "+name, func(data []byte) ([]byte, error) {
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
	yamlData, err := s.ds.getData("get-envs", s.stateFilePath)
	if err != nil {
		return nil, err
	}

	ds := &yamlDataStore{}
	return ds.load(ctx, yamlData)
}

type gitDataStore struct {
	auth transport.AuthMethod

	// gitRepoURL is the URL of the git repository that contains the gitops config
	// It needs to be a URL that can be handled by go-git and git-clone.
	gitRepoURL string

	// repository is the local git repository that contains the gitops config
	// It is cloned from gitRepoURL.
	// It is not nil only after Clone() has succeeded.
	repository *git.Repository

	// authorUserName is the username to be used when committing changes to the gitops config
	// It is usually the name of the bot user, with or without an email address,
	// in the form "username <email>".
	authorUserName string

	// baseRefName is the name of the branch that contains the gitops config
	// It is usually "master" or "main".
	baseRefName plumbing.ReferenceName

	// gitRoot is the root of the local git repository, used to
	// clone and checkout the remote repository that contains the gitops config
	// or the kustomize config we are going to modify.
	// If empty, we will use in-memory filesystem.
	gitRoot string
	// cloned is true when the git repository has been cloned.
	cloned bool
}

func (s *gitDataStore) getData(branch, path string) ([]byte, error) {
	w, err := s.createAndCheckoutNewBranch(branch)
	if err != nil {
		return nil, err
	}

	f, err := w.Filesystem.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %q: %w", path, err)
	}

	yamlData, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %q: %w", path, err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("unable to close file %q: %w", path, err)
	}

	return yamlData, nil
}

func (s *gitDataStore) modifyState(branch, path, message string, fn func([]byte) ([]byte, error)) error {
	w, err := s.createAndCheckoutNewBranch(branch)
	if err != nil {
		return err
	}

	err = s.modifyAndAdd(w, path, fn)
	if err != nil {
		return fmt.Errorf("unable to modify and add file %q: %w", path, err)
	}

	err = s.verify(w)
	if err != nil {
		return fmt.Errorf("unable to verify git status: %w", err)
	}

	hash, err := w.Commit(
		message,
		&git.CommitOptions{
			Author: &object.Signature{
				Name:  s.authorUserName,
				Email: "",
				When:  time.Now(),
			},
		})
	if err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}

	ref := plumbing.NewReferenceFromStrings(branch, hash.String())
	if err := s.repository.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("unable to set reference %v: %w", ref, err)
	}

	remote, err := s.repository.Remote("origin")
	if err != nil {
		return fmt.Errorf("unable to get remote origin: %w", err)
	}

	refSpec := config.RefSpec(plumbing.ReferenceName(branch) + ":" + plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)))
	if err := remote.Push(&git.PushOptions{
		Progress: os.Stdout,
		RefSpecs: []config.RefSpec{
			refSpec,
		},
		Auth: s.auth,
	}); err != nil {
		return fmt.Errorf("unable to push %v to remote origin: %w", refSpec, err)
	}

	return nil
}

func (s *gitDataStore) Clone() error {
	var (
		storage storage.Storer
		fs      billy.Filesystem
	)
	if s.gitRoot != "" {
		fs = osfs.New(s.gitRoot)
		storage = filesystem.NewStorage(
			osfs.New(filepath.Join(s.gitRoot, ".git")),
			cache.NewObjectLRUDefault(),
		)
	} else {
		storage = memory.NewStorage()
		fs = memfs.New()
	}
	r, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:  s.gitRepoURL,
		Auth: s.auth,
	})
	s.repository = r

	return err
}

func (s gitDataStore) DeleteBranch(branch string) (err error) {
	return s.repository.Storer.RemoveReference(plumbing.ReferenceName(branch))
}

func (s gitDataStore) createAndCheckoutNewBranch(branch string) (*git.Worktree, error) {
	if !s.cloned {
		if err := s.Clone(); err != nil {
			return nil, err
		}

		s.cloned = true
	}

	if err := s.DeleteBranch(branch); err != nil {
		fmt.Printf("Unable to delete branch %q: %v", branch, err)
	}

	w, err := s.repository.Worktree()
	if err != nil {
		return nil, err
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: s.baseRefName,
	}); err != nil {
		return nil, fmt.Errorf("unable to checkout branch %q: %w", s.baseRefName, err)
	}

	if err := w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       s.auth,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("unable to pull from remote origin: %w", err)
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.ReferenceName(branch),
	}); err != nil {
		return nil, fmt.Errorf("unable to checkout branch %q: %w", branch, err)
	}

	return w, nil
}

func (s gitDataStore) verify(w *git.Worktree) error {
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("unable to run git-status: %w", err)
	}

	for path, status := range status {
		switch status.Staging {
		case git.Modified | git.Added | git.Deleted:
		default:
			return fmt.Errorf("failed to verify git status: all files should be modified. File: %v %s", status, path)
		}
	}

	return nil
}

func (s gitDataStore) modifyAndAdd(w *git.Worktree, path string, fn func([]byte) ([]byte, error)) error {
	if _, err := w.Filesystem.Stat(path); err != nil {
		return fmt.Errorf("unable to stat file %q: %w", path, err)
	}

	file, err := w.Filesystem.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %w", path, err)
	}

	b, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("unable to read file %q: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("unable to close file %q: %w", path, err)
	}

	data, err := fn(b)
	if err != nil {
		return fmt.Errorf("unable to run fn: %w", err)
	}

	if err := w.Filesystem.Remove(path); err != nil {
		return fmt.Errorf("unable to remove file %q: %w", path, err)
	}

	file, err = w.Filesystem.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("unable to create file %q: %w", path, err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("unable to write file %q: %w", path, err)
	}

	if _, err := file.Write([]byte("\n")); err != nil {
		return fmt.Errorf("unable to write new-line to file %q: %w", path, err)
	}

	_, err = w.Add(path)
	if err != nil {
		return fmt.Errorf("unable to run git-add: %w", err)
	}

	return nil
}
