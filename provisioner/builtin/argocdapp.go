package builtin

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/builtin/k8sdeploy"
	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/mumoshu/prenv/render"
)

type BuiltinArgoCDAppProvisioner struct {
	Config    config.ArgoCDApp
	EnvParams config.EnvArgs
}

func (p *BuiltinArgoCDAppProvisioner) Render(ctx context.Context, dir string) (*plugin.RenderResult, error) {
	a, err := generateOne(p.EnvParams, "", p.Config)
	if err != nil {
		return nil, err
	}

	t := render.Template{
		Name: a.Name,
		Body: k8sdeploy.TemplateArgoCDApp,
		Data: a,
	}

	r, err := render.ToDir(dir, t)
	if err != nil {
		return nil, err
	}

	return &plugin.RenderResult{
		AddedOrModifiedFiles: r,
	}, nil
}

func (p *BuiltinArgoCDAppProvisioner) Apply(ctx context.Context, r *plugin.RenderResult) (*plugin.Result, error) {
	if err := k8sdeploy.KubectlApply(ctx, "."); err != nil {
		return nil, fmt.Errorf("unable to apply Kubernetes resources: %w", err)
	}

	return &plugin.Result{}, nil
}

func (p *BuiltinArgoCDAppProvisioner) Destroy(ctx context.Context) (*plugin.Result, error) {
	if err := k8sdeploy.KubectlDelete(ctx, "."); err != nil {
		return nil, fmt.Errorf("unable to delete Kubernetes resources: %w", err)
	}

	return &plugin.Result{}, nil
}

func generateOne(env config.EnvArgs, shortName string, ac config.ArgoCDApp) (*k8sdeploy.AppParams, error) {
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
