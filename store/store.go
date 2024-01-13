// Store is an interface for storing configs.
// There are two implementations of this interface:
// - Local
// - Git
// - PullRequest
package store

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/envvar"
	"github.com/mumoshu/prenv/provisioner/plugin"
)

type Store interface {
	Put(context context.Context, path string, content string) error
	List(context context.Context, path string) ([]string, error)
	Get(context context.Context, path string) (*string, error)
	Delete(context context.Context, path string) error

	// Transact runs the given function within the directory that the store stores the configs.
	//
	// The path argument is the path to the directory.
	// It can be a temporary directory or the specified directory in the clone of the gitops repository,
	// or whatever the store implementation relies on.
	//
	// The function is expected to return the result of the rendering, which contains
	// added, modified, and deleted files.
	// The function is expected to return an error if the rendering fails.
	//
	// The store implementation is expected to include the files returned by the function
	// in the commit.
	// The caller is expected to call Commit after calling Transact.
	Transact(fn func(path string) (*plugin.RenderResult, error)) (*plugin.RenderResult, error)

	// Commit commits the changes made to the store.
	// The subject and body are used as the commit message, if applicable.
	// If the store does not support commits, it returns nil.
	Commit(context context.Context, subject, body string) error
}

// Init inits file store based on the given config.Delegate.
func Init(id string, t time.Time, d *config.Delegate) Store {
	if d == nil {
		return newLocal(id)
	}

	var repoURL string
	if strings.Count(d.Git.Repo, "/") == 1 {
		githubBaseURL := "https://github.com/"
		if os.Getenv(envvar.GitHubEnterpriseURL) != "" {
			githubBaseURL = os.Getenv(envvar.GitHubEnterpriseURL)
		}
		repoURL = githubBaseURL + d.Git.Repo + ".git"
	} else if strings.Count(d.Git.Repo, "/") == 2 {
		repoURL = "https://" + d.Git.Repo + ".git"
	} else if strings.HasPrefix(d.Git.Repo, "https://") {
		repoURL = d.Git.Repo
	} else {
		panic(fmt.Sprintf("invalid repo in prenv.yaml: %s", d.Git.Repo))
	}

	baseBranch := os.Getenv(envvar.BaseBranch)
	if d.Git.Branch != "" {
		baseBranch = d.Git.Branch
	}

	auth := &http.BasicAuth{
		Username: "prenvbot", // This can be anything except an empty string
		Password: os.Getenv(envvar.GitHubToken),
	}

	var newBranch string

	if d.PullRequest != nil {
		newBranch = fmt.Sprintf("prenv/%s-%s", id, t.Format("20060102150405"))
	}

	gitRoot := os.Getenv(envvar.GitRoot)
	if gitRoot == "" {
		gitRoot = ".prenv/repositories"
	}

	g := newGit(
		auth,
		baseBranch,
		newBranch,
		repoURL,
		os.Getenv(envvar.GitCommitAuthorUserName),
		os.Getenv(envvar.GitCommitAuthorEmail),
		gitRoot,
		d.Git.Push,
	)

	if d.PullRequest != nil {
		return &PullRequest{
			RepositoryURL: repoURL,
			Git:           g,
			PullRequest:   d.PullRequest,
		}
	}

	return g
}
