package config

import (
	"context"
	"net/url"
	"os"

	"github.com/google/go-github/v56/github"
	"github.com/mumoshu/prenv/envvar"
	"golang.org/x/oauth2"
)

func NewGitHubClient() *github.Client {
	token := os.Getenv(envvar.GitHubToken)

	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
	client := github.NewClient(httpClient)

	if u := os.Getenv(envvar.GitHubBaseURL); u != "" {
		u, err := url.Parse(u)
		if err != nil {
			panic(err)
		}

		client.BaseURL = u
	}

	return client
}
