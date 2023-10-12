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
package env

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/infra"
	"github.com/mumoshu/prenv/k8sdeploy"
	"github.com/mumoshu/prenv/state"
)

func Apply(ctx context.Context, cfg config.Config) error {
	store := &state.Store{}
	envNameTemplate := cfg.Environment.NameTemplate
	if envNameTemplate == "" {
		envNameTemplate = "prenv-{{.PullRequestNumber}}"
	}
	a := &k8sdeploy.ArgoCDApp{}
	if err := a.LoadEnvVars(); err != nil {
		return err
	}
	envNameTmpl := template.Must(template.New("envName").Parse(envNameTemplate))
	var buf bytes.Buffer
	if err := envNameTmpl.Execute(&buf, a); err != nil {
		return err
	}
	envName := buf.String()
	if err := store.AddEnvironmentName(ctx, envName); err != nil {
		return err
	}

	// Add the new SQS queue and reconfigures the SQS forwarder.
	if err := infra.Reconcile(ctx, cfg); err != nil {
		return err
	}

	// Deploy the Kubernetes resources.
	if err := deployKubernetesResources(ctx, *a); err != nil {
		return err
	}

	return nil
}

func deployKubernetesResources(ctx context.Context, app k8sdeploy.ArgoCDApp) error {
	name := fmt.Sprintf("%s-%d", app.ArgoCDApp.Name, app.PullRequestNumber)

	if err := k8sdeploy.Apply(ctx,
		k8sdeploy.M{
			Name:         name,
			Template:     k8sdeploy.TemplateArgoCDApp,
			TemplateData: app,
		},
	); err != nil {
		return fmt.Errorf("unable to deploy Kubernetes resources: %w", err)
	}

	return nil
}

func Destroy(ctx context.Context, cfg config.Config) error {
	store := &state.Store{}
	envNameTemplate := cfg.Environment.NameTemplate
	if envNameTemplate == "" {
		envNameTemplate = "prenv-{{.PullRequestNumber}}"
	}
	a := &k8sdeploy.ArgoCDApp{}
	if err := a.LoadEnvVars(); err != nil {
		return err
	}
	envNameTmpl := template.Must(template.New("envName").Parse(envNameTemplate))
	var buf bytes.Buffer
	if err := envNameTmpl.Execute(&buf, a); err != nil {
		return err
	}
	envName := buf.String()
	if err := store.DeleteEnvironmentName(ctx, envName); err != nil {
		return err
	}

	// Delete the Kubernetes resources.
	if err := destroyKubernetesResources(ctx, *a); err != nil {
		return err
	}

	// Delete the SQS queue and reconfigures the SQS forwarder.
	if err := infra.Reconcile(ctx, cfg); err != nil {
		return err
	}

	return nil
}

func destroyKubernetesResources(ctx context.Context, app k8sdeploy.ArgoCDApp) error {
	name := fmt.Sprintf("%s-%d", app.ArgoCDApp.Name, app.PullRequestNumber)

	if err := k8sdeploy.Delete(ctx,
		k8sdeploy.M{
			Name:         name,
			Template:     k8sdeploy.TemplateArgoCDApp,
			TemplateData: app,
		},
	); err != nil {
		return fmt.Errorf("unable to delete Kubernetes resources: %w", err)
	}

	return nil
}
