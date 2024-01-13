// The env package is responsible for creating and destroying the pull-request environment.
// It has two major functions:
// 1. Create the pull-request environment
// 2. Destroy the pull-request environment
//
// Both of them are implemented as a subcommand of the `prenv` command.
//
// Each command requires the following environment variables:
// - GITHUB_SHA
// - GITHUB_EVENT_PATH
// - GITHUB_TOKEN
//
// _SHA is the SHA of the commit to be deployed and used as the tag of the docker image.
//
// _EVENT_PATH is the path to the file that contains the event payload, which is present
// when the command is invoked by GitHub Actions.
// This package loads the payload, and extracts the pull request number from it.
//
// This package manages both the lifecycle of per-environment SQS queues and the Kubernetes resources,
// and reconfiguration of the SQS forwarder.
package generator

import (
	"bytes"
	"text/template"

	"github.com/mumoshu/prenv/config"
)

func BuildGitHubActionsPullRequestEnvArgs(cfg config.Config) (*config.EnvArgs, error) {
	envNameTemplate := cfg.EnvironmentNameTemplate
	if envNameTemplate == "" && cfg.NamePrefix != "" {
		envNameTemplate = "{{ .NamePrefix }}{{.PullRequest.Number}}"
	} else {
		envNameTemplate = "prenv-{{.PullRequest.Number}}"
	}

	envParams := config.EnvArgs{}
	if err := envParams.LoadEnvVarsAndEvent(); err != nil {
		return nil, err
	}

	type templateData struct {
		config.Config
		*config.EnvArgs
	}
	d := templateData{
		Config:  cfg,
		EnvArgs: &envParams,
	}

	envNameTmpl := template.Must(template.New("envName template").Parse(envNameTemplate))
	var buf bytes.Buffer
	if err := envNameTmpl.Execute(&buf, d); err != nil {
		return nil, err
	}
	envParams.Name = buf.String()

	return &envParams, nil
}
