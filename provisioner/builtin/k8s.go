package builtin

import (
	"context"
	"fmt"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/builtin/k8sdeploy"
	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/mumoshu/prenv/render"
)

type BuiltinKubernetesProvisioner struct {
	Config config.KubernetesResources
}

func (p *BuiltinKubernetesProvisioner) Apply(ctx context.Context, r *plugin.RenderResult) (*plugin.Result, error) {
	cfg := p.Config

	if err := deployKubernetesResources(ctx, cfg); err != nil {
		return nil, fmt.Errorf("unable to deploy Kubernetes resources: %w", err)
	}

	return &plugin.Result{}, nil
}

func (p *BuiltinKubernetesProvisioner) Destroy(ctx context.Context) (*plugin.Result, error) {
	return &plugin.Result{}, nil
}

func (p *BuiltinKubernetesProvisioner) Render(ctx context.Context, dir string) (*plugin.RenderResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func deployKubernetesResources(ctx context.Context, k8sRes config.KubernetesResources) error {
	defaults := config.KubernetesApp{
		Namespace: "prenv",
		Image:     k8sRes.Image,
	}

	sf, err := k8sRes.SQSForwarder.BuildDeployConfig(defaults)
	if err != nil {
		return fmt.Errorf("unable to build deploy config for sqs forwarder: %w", err)
	}
	ow, err := k8sRes.OutgoingWebhook.BuildDeployConfig(defaults)
	if err != nil {
		return fmt.Errorf("unable to build deploy config for outgoing webhook: %w", err)
	}

	if err := k8sdeploy.Apply(ctx,
		render.Template{
			Name: sf.Name,
			Body: k8sdeploy.TemplateDeployment,
			Data: sf,
		},
		render.Template{
			Name: ow.Name,
			Body: k8sdeploy.TemplateDeployment,
			Data: ow,
		},
	); err != nil {
		return fmt.Errorf("unable to apply Kubernetes manifests: %w", err)
	}

	return nil
}
