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

	envParams, err := generateEnvParams(cfg)
	if err != nil {
		return err
	}

	if err := store.AddEnvironmentName(ctx, envParams.Name); err != nil {
		return err
	}

	// Add the new SQS queue and reconfigures the SQS forwarder.
	if err := infra.Reconcile(ctx, cfg); err != nil {
		return err
	}

	as, err := generateManyK8sApps(*envParams, cfg)
	if err != nil {
		return err
	}

	for _, a := range as {
		// Deploy the Kubernetes resources.
		if err := deployKubernetesResources(ctx, *a); err != nil {
			return err
		}
	}

	return nil
}

func deployKubernetesResources(ctx context.Context, app k8sdeploy.AppParams) error {
	if err := k8sdeploy.Apply(ctx,
		k8sdeploy.M{
			Name:         app.Name,
			Template:     k8sdeploy.TemplateArgoCDApp,
			TemplateData: app,
		},
	); err != nil {
		return fmt.Errorf("unable to deploy Kubernetes resources: %w", err)
	}

	return nil
}

func generateEnvParams(cfg config.Config) (*k8sdeploy.EnvParams, error) {
	envNameTemplate := cfg.EnvironmentNameTemplate
	if envNameTemplate == "" {
		envNameTemplate = "{{ .BaseName }}-{{.PullRequestNumber}}"
	}

	envParams := k8sdeploy.EnvParams{}
	if err := envParams.LoadEnvVarsAndEvent(); err != nil {
		return nil, err
	}

	envNameTmpl := template.Must(template.New("envName").Parse(envNameTemplate))
	var buf bytes.Buffer
	if err := envNameTmpl.Execute(&buf, envParams); err != nil {
		return nil, err
	}
	envParams.Name = buf.String()

	return &envParams, nil
}

func Destroy(ctx context.Context, cfg config.Config) error {
	store := &state.Store{}

	envParams, err := generateEnvParams(cfg)
	if err != nil {
		return err
	}

	if err := store.DeleteEnvironmentName(ctx, envParams.Name); err != nil {
		return err
	}

	as, err := generateManyK8sApps(*envParams, cfg)
	if err != nil {
		return err
	}

	for _, a := range as {
		if err := destroyOneK8sApp(ctx, *a); err != nil {
			return fmt.Errorf("destroying %q: %w", a.Name, err)
		}
	}

	// Delete the SQS queue and reconfigures the SQS forwarder.
	if err := infra.Reconcile(ctx, cfg); err != nil {
		return err
	}

	return nil
}

func destroyOneK8sApp(ctx context.Context, a k8sdeploy.AppParams) error {
	// Delete the Kubernetes resources.
	if err := destroyKubernetesResources(ctx, a); err != nil {
		return err
	}

	return nil
}

func destroyKubernetesResources(ctx context.Context, app k8sdeploy.AppParams) error {
	if err := k8sdeploy.Delete(ctx,
		k8sdeploy.M{
			Name:         app.Name,
			Template:     k8sdeploy.TemplateArgoCDApp,
			TemplateData: app,
		},
	); err != nil {
		return fmt.Errorf("unable to delete Kubernetes resources: %w", err)
	}

	return nil
}

func generateManyK8sApps(env k8sdeploy.EnvParams, cfg config.Config) ([]*k8sdeploy.AppParams, error) {
	var as []*k8sdeploy.AppParams

	if len(cfg.Services) == 0 && cfg.ArgoCDApp == nil {
		return nil, fmt.Errorf("services or argocdApp is required")
	} else if len(cfg.Services) > 0 && cfg.ArgoCDApp != nil {
		return nil, fmt.Errorf("services and argocdApp are mutually exclusive")
	}

	if cfg.ArgoCDApp != nil {
		if env.AppNameTemplate == "" {
			env.AppNameTemplate = "{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}"
		}

		a, err := generateOne(env, "", *cfg.ArgoCDApp)
		if err != nil {
			return nil, err
		}

		as = append(as, a)

		return as, nil
	}

	if env.AppNameTemplate == "" {
		env.AppNameTemplate = "{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}-{{ .ShortName }}"
	}

	for shortName, svc := range cfg.Services {
		ac := svc.ArgoCDApp
		if shortName == "" {
			return nil, fmt.Errorf("services.%s.argocdApp: shortName is required: encountered %q", shortName, shortName)
		}
		a, err := generateOne(env, shortName, ac)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}

	return as, nil
}

func generateOne(env k8sdeploy.EnvParams, shortName string, ac config.ArgoCDApp) (*k8sdeploy.AppParams, error) {
	a := &k8sdeploy.AppParams{
		ShortName:   shortName,
		ArgoCDApp:   ac,
		Environment: env,
	}

	if env.AppNameTemplate == "" {
		return nil, fmt.Errorf("assertion error: environment.appNameTemplate is required")
	}

	appNameTmpl := template.Must(template.New("appName").Parse(env.AppNameTemplate))
	var buf bytes.Buffer
	if err := appNameTmpl.Execute(&buf, a); err != nil {
		return nil, err
	}
	appName := buf.String()

	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("invalid argocdApp: %w", err)
	}

	a.Name = appName

	return a, nil
}
