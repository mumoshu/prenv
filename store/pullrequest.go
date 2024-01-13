package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v56/github"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/plugin"
)

type PullRequest struct {
	RepositoryURL string
	Git           *Git
	PullRequest   *config.PullRequest
}

func (c *PullRequest) Transact(fn func(path string) (*plugin.RenderResult, error)) (*plugin.RenderResult, error) {
	return c.Git.Transact(fn)
}

func (c *PullRequest) Put(ctx context.Context, path string, content string) error {
	return fmt.Errorf("not implemented")
}

func (c *PullRequest) List(ctx context.Context, path string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *PullRequest) Get(ctx context.Context, path string) (*string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *PullRequest) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("not implemented")
}

func (c *PullRequest) Commit(ctx context.Context, subject, body string) error {
	if err := c.Git.Commit(ctx, subject, body); err != nil {
		return err
	}

	return c.createPullRequest(ctx, subject, body)
}

func (c *PullRequest) createPullRequest(ctx context.Context, subject, body string) error {
	client := config.NewGitHubClient()

	split := strings.Split(c.RepositoryURL, "/")

	owner := split[len(split)-2]
	repo := split[len(split)-1]

	if strings.HasSuffix(repo, ".git") {
		repo = repo[:len(repo)-len(".git")]
	}

	_, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(subject),
		Head:  github.String(string(*c.Git.NewRefName)),
		Base:  github.String(string(c.Git.BaseRefName)),
		Body:  github.String(body),
	})
	if err != nil {
		return err
	}

	return nil
}
