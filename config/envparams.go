package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v56/github"
	"github.com/mumoshu/prenv/envvar"
)

// EnvArgs is the parameters for the environment to be deployed per pull request.
// This contains the environment-generator-specific arguments
// that is used to generate the environment-specific configuration.
type EnvArgs struct {
	// Name is the name of the environment.
	// It will be NameBase-PullRequestNumber by default.
	Name string

	// AppNameTemplate is the Go template used to generate the name of the ArgoCD application.
	// It is `{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}` or `{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}-{{ .ShortName }} by default,
	AppNameTemplate string

	// The following fields are set by LoadEnvVars.
	PullRequest *PullRequestEnvArgs `yaml:"pullRequest,omitempty"`
}

func (a *EnvArgs) LoadEnvVarsAndEvent() error {
	pr := &PullRequestEnvArgs{}
	if err := pr.LoadEnvVarsAndEvent(); err != nil {
		return err
	}

	a.PullRequest = pr

	return nil
}

func (a *EnvArgs) Validate() error {
	if err := a.PullRequest.Validate(); err != nil {
		return err
	}

	return nil
}

type PullRequestEnvArgs struct {
	// Number is the number of the pull request to be deployed.
	Number int `yaml:"number,omitempty"`
	// HeadSHA is the SHA of the head commit of the pull request to be deployed.
	HeadSHA string `yaml:"headSHA,omitempty"`

	// Numbers is numbers of all the open pull requests.
	Numbers []int `yaml:"pullRequestNumbers,omitempty"`

	// Repository is the repository that prenv is originally triggered from.
	// It is in the form of owner/repo.
	// This is used to populate PullRequestNumbers.
	Repository string `yaml:"repository,omitempty"`
}

// LoadEnvVarsAndEvent loads the environment variables and the GitHub Actions event payload.
// The loaded values are set to the EnvParams and therefore avaiable for Go templates used
// for generating the Kubernetes manifests.
func (a *PullRequestEnvArgs) LoadEnvVarsAndEvent() error {
	prNumber, err := GetPullRequestNumber()
	if err != nil {
		return err
	}

	sha, err := GetSHA()
	if err != nil {
		return err
	}

	a.HeadSHA = sha

	if prNumber != nil {
		a.Number = *prNumber
	}

	a.Repository = os.Getenv(envvar.GitHubRepository)

	_, err = GetEventPayload()
	if err != nil {
		return err
	}

	if err := a.LoadPullRequestNumbers(); err != nil {
		return err
	}

	return nil
}

func (a *PullRequestEnvArgs) LoadPullRequestNumbers() error {
	if a.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	ownerRepo := strings.Split(a.Repository, "/")
	if len(ownerRepo) != 2 {
		return fmt.Errorf("repository must be in the form of owner/repo")
	}

	owner := ownerRepo[0]
	repo := ownerRepo[1]

	client := NewGitHubClient()

	var prNums []int

	r, _, err := client.PullRequests.List(context.Background(), owner, repo, &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		return err
	}

	for _, pr := range r {
		prNums = append(prNums, *pr.Number)
	}

	a.Numbers = prNums

	return nil
}

func (a *PullRequestEnvArgs) Validate() error {
	if a.HeadSHA == "" {
		return fmt.Errorf("githubSHA is required. Set GITHUB_SHA env var")
	}

	if a.Number == 0 {
		return fmt.Errorf("pullRequestNumber is required")
	}

	return nil
}
