package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/mumoshu/prenv/provisioner/plugin"
)

// Git is a key-value-store-like interface for gitops config repository.
type Git struct {
	Auth transport.AuthMethod

	// GitRepoURL is the URL of the git repository that contains the gitops config
	// It needs to be a URL that can be handled by go-git and git-clone.
	GitRepoURL string

	// repository is the local git repository that contains the gitops config
	// It is cloned from gitRepoURL.
	// It is not nil only after Clone() has succeeded.
	repository *git.Repository

	// worktree is the local git worktree that contains the gitops config
	worktree *git.Worktree

	// AuthorName is the username to be used when committing changes to the gitops config
	// It is usually the name of the bot user, with or without an email address,
	// in the form "username", not "username <email>".
	AuthorName string

	AuthorEmail string

	baseBranch string
	// BaseRefName is the name of the branch that contains the gitops config
	// It is usually "master" or "main".
	BaseRefName plumbing.ReferenceName

	newBranch  string
	NewRefName *plumbing.ReferenceName

	// GitRoot is the root of the local git repository, used to
	// clone and checkout the remote repository that contains the gitops config
	// or the kustomize config we are going to modify.
	// If empty, we will use in-memory filesystem.
	GitRoot string
	// cloned is true when the git repository has been cloned.
	cloned bool

	// Push specifies whether the gitops config is updated via git push.
	Push bool
}

func newGit(auth transport.AuthMethod, baseBranch, newBranch, gitRepoURL, authorUserName, authorEmail, gitRoot string, push bool) *Git {
	baseRefName := plumbing.Master
	if baseBranch != "" {
		baseRefName = plumbing.NewBranchReferenceName(baseBranch)
	}

	g := &Git{
		Auth:        auth,
		baseBranch:  baseBranch,
		BaseRefName: baseRefName,
		GitRepoURL:  gitRepoURL,
		AuthorName:  authorUserName,
		AuthorEmail: authorEmail,
		GitRoot:     gitRoot,
		Push:        push,
	}

	if newBranch != "" {
		g.newBranch = newBranch

		n := plumbing.ReferenceName("refs/heads/" + newBranch)
		g.NewRefName = &n
	}

	return g
}

func (g *Git) Transact(fn func(path string) (*plugin.RenderResult, error)) (*plugin.RenderResult, error) {
	w, err := g.createAndCheckoutNewBranch("")
	if err != nil {
		var msg string
		if g.repository != nil {
			branches, err := g.repository.Branches()
			if err != nil {
				return nil, fmt.Errorf("unable to get branches: %w", err)
			}

			var bs []string
			if err := branches.ForEach(func(b *plumbing.Reference) error {
				bs = append(bs, b.Name().String())
				return nil
			}); err != nil {
				return nil, fmt.Errorf("unable to iterate over branches: %w", err)
			}

			msg = fmt.Sprintf("branches: %v", branches)
		}

		return nil, fmt.Errorf("unable to create and/or checkout branch: %w: %s", err, msg)
	}

	r, err := fn(g.getLocalRepoPath())
	if err != nil {
		return nil, err
	}

	for _, f := range r.AddedOrModifiedFiles {
		if _, err := w.Add(f); err != nil {
			return nil, fmt.Errorf("unable to run git-add (chroot=%s, name=%s): %w", g.getLocalRepoPath(), f, err)
		}
	}

	for _, f := range r.DeletedFiles {
		if _, err := w.Remove(f); err != nil {
			return nil, fmt.Errorf("unable to run git-rm: %w", err)
		}
	}

	return r, nil
}

func (g *Git) Get(ctx context.Context, path string) (*string, error) {
	// TODO get file from the workspace
	return nil, nil
}

func (g *Git) Put(ctx context.Context, path string, content string) error {
	// TODO put file and run git-add
	return nil
}

func (g *Git) List(ctx context.Context, path string) ([]string, error) {
	// TODO list files under the workspace
	return nil, nil
}

func (g *Git) Delete(ctx context.Context, path string) error {
	// TODO run git-rm
	return nil
}

func (g *Git) Commit(ctx context.Context, subject, body string) error {
	if !g.Push {
		return nil
	}

	w, err := g.getWorktree()
	if err != nil {
		return fmt.Errorf("unable to get worktree: %w", err)
	}

	hash, err := w.Commit(subject, &git.CommitOptions{
		Author: &object.Signature{
			Name:  g.AuthorName,
			Email: g.AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}

	ref := plumbing.NewReferenceFromStrings(string(g.BaseRefName), hash.String())
	if err := g.repository.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("unable to set reference %v: %w", ref, err)
	}

	remote, err := g.repository.Remote("origin")
	if err != nil {
		return fmt.Errorf("unable to get remote origin: %w", err)
	}

	if !g.Push {
		return nil
	}

	var refName plumbing.ReferenceName
	if g.NewRefName == nil {
		refName = g.BaseRefName
	} else {
		refName = *g.NewRefName
	}

	if err := remote.Push(&git.PushOptions{
		Progress: os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(refName + ":" + refName),
		},
		Auth: g.Auth,
	}); err != nil {
		return fmt.Errorf("unable to push %v to remote origin: %w", *g.NewRefName, err)
	}

	return nil
}

func (g *Git) getWorktree() (*git.Worktree, error) {
	if g.worktree != nil {
		return g.worktree, nil
	}

	w, err := g.repository.Worktree()
	if err != nil {
		return nil, err
	}

	g.worktree = w

	return w, nil
}

func (s *Git) GetFileFromBranch(branch, path string) ([]byte, error) {
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

func (s *Git) ModifyFile(branch, path, message string, fn func([]byte) ([]byte, error)) error {
	return s.ModifyWorktree(branch, message, func(w *git.Worktree) error {
		if err := s.modifyFile(w.Filesystem, path, fn); err != nil {
			return fmt.Errorf("unable to modify and add file %q: %w", path, err)
		}

		if _, err := w.Add(path); err != nil {
			return fmt.Errorf("unable to run git-add: %w", err)
		}

		return nil
	})
}

// ModifyDir modifies the files in the git worktree.
// It does so by checking out a new feature branch from the base branch,
// modifying the files in the worktree, and committing the changes to the feature branch.
// The modification is delegated to the fn function.
//
// The fn function receives the path of the directory to be modified,
// and a filesystem that is guaranteed to have the directory specified by the path.
// In other words, it's this function's responsibility to create the directory
// specified by the path if it does not exist.
//
// The fn function does not need to add the modified files to the git index,
// as this function will do it automatically.
func (s *Git) ModifyDir(branch, path, message string, fn func(string, billy.Filesystem) error) error {
	return s.ModifyWorktree(branch, message, func(w *git.Worktree) error {
		fs := w.Filesystem

		if err := fs.MkdirAll(path, 0777); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", path, err)
		}

		if err := fn(path, fs); err != nil {
			return fmt.Errorf("unable to run fn: %w", err)
		}

		if _, err := w.Add(path); err != nil {
			return fmt.Errorf("unable to run git-add: %w", err)
		}

		return nil
	})
}

// ModifyWorktree modifies the files in the git worktree.
// It does so by checking out a new feature branch from the base branch,
// modifying the files in the worktree, and committing the changes to the feature branch.
//
// The modification is delegated to the fn function.
// It's the responsibility of the fn function to modify the files in the worktree,
// and to add the modified files to the git index.
func (s *Git) ModifyWorktree(branch, message string, fn func(*git.Worktree) error) error {
	w, err := s.createAndCheckoutNewBranch(branch)
	if err != nil {
		return err
	}

	if err := fn(w); err != nil {
		return err
	}

	err = s.verify(w)
	if err != nil {
		return fmt.Errorf("unable to verify git status: %w", err)
	}

	hash, err := w.Commit(
		message,
		&git.CommitOptions{
			Author: &object.Signature{
				Name:  s.AuthorName,
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
		Auth: s.Auth,
	}); err != nil {
		return fmt.Errorf("unable to push %v to remote origin: %w", refSpec, err)
	}

	return nil
}

func (s *Git) getLocalRepoPath() string {
	dir := s.GitRepoURL
	dir = strings.TrimPrefix(dir, "https://")
	dir = strings.TrimPrefix(dir, "http://")
	dir = strings.TrimPrefix(dir, "git@")
	dir = strings.TrimSuffix(dir, ".git")

	return filepath.Join(s.GitRoot, dir)
}

func (s *Git) clone() error {
	var (
		storage storage.Storer
		fs      billy.Filesystem
	)

	if s.GitRoot != "" {
		gitRoot := s.getLocalRepoPath()
		fs = osfs.New(gitRoot)
		storage = filesystem.NewStorage(
			osfs.New(filepath.Join(gitRoot, ".git")),
			cache.NewObjectLRUDefault(),
		)
	} else {
		storage = memory.NewStorage()
		fs = memfs.New()
	}
	r, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:  s.GitRepoURL,
		Auth: s.Auth,
	})

	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		r, err = git.PlainOpen(s.getLocalRepoPath())
		if err != nil {
			return fmt.Errorf("unable to open local git repository: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to clone git repository %s: %w", s.GitRepoURL, err)
	}

	s.repository = r

	return nil
}

func (s *Git) deleteBranch(branch string) (err error) {
	return s.repository.Storer.RemoveReference(plumbing.ReferenceName(branch))
}

func (s *Git) createAndCheckoutNewBranch(branch string) (*git.Worktree, error) {
	if !s.cloned {
		if err := s.clone(); err != nil {
			return nil, err
		}

		s.cloned = true
	}

	// if err := s.deleteBranch(branch); err != nil {
	// 	fmt.Printf("Unable to delete branch %q: %v", branch, err)
	// }

	w, err := s.getWorktree()
	if err != nil {
		return nil, fmt.Errorf("unable to get worktree: %w", err)
	}

	if checkoutErr := w.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: s.BaseRefName,
	}); checkoutErr != nil {
		remote, err := s.repository.Remote("origin")
		if err != nil {
			return nil, fmt.Errorf("unable to get remote origin: %w", err)
		}

		if err := remote.Fetch(&git.FetchOptions{
			Auth: s.Auth,
			RefSpecs: []config.RefSpec{
				config.RefSpec(s.BaseRefName + ":" + s.BaseRefName),
			},
		}); err != nil {
			return nil, fmt.Errorf("unable to checkout %s: %w\nunable to fetch from remote origin: %w", s.BaseRefName, checkoutErr, err)
		}

		if err := w.Checkout(&git.CheckoutOptions{
			Create: false,
			Branch: s.BaseRefName,
		}); err != nil {
			return nil, fmt.Errorf("unable to checkout branch %q: %w", s.BaseRefName, err)
		}
	}

	if err := w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       s.Auth,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("unable to pull from remote origin: %w", err)
	}

	var b *plumbing.ReferenceName

	if branch != "" {
		n := plumbing.ReferenceName(branch)
		b = &n
	} else if s.NewRefName != nil {
		b = s.NewRefName
	}

	if b != nil {
		// h, err := s.repository.Head()
		// if err != nil {
		// 	return nil, fmt.Errorf("unable to resolve revision %q: %w", s.BaseRefName, err)
		// }
		if err := w.Checkout(&git.CheckoutOptions{
			Create: true,
			// Hash:   h.Hash(),
			Branch: *b,
		}); err != nil {
			return nil, fmt.Errorf("unable to checkout branch %q: %w", *b, err)
		}
	}

	return w, nil
}

func (s *Git) verify(w *git.Worktree) error {
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

func (s *Git) modifyFile(fs billy.Filesystem, path string, fn func([]byte) ([]byte, error)) error {
	if _, err := fs.Stat(path); err != nil {
		return fmt.Errorf("unable to stat file %q: %w", path, err)
	}

	file, err := fs.Open(path)
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

	if err := fs.Remove(path); err != nil {
		return fmt.Errorf("unable to remove file %q: %w", path, err)
	}

	file, err = fs.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("unable to create file %q: %w", path, err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("unable to write file %q: %w", path, err)
	}

	if _, err := file.Write([]byte("\n")); err != nil {
		return fmt.Errorf("unable to write new-line to file %q: %w", path, err)
	}

	return nil
}
