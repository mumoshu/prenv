package provisioner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/builtin"
	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/mumoshu/prenv/provisioner/render"
	"github.com/mumoshu/prenv/store"
)

type PluginConfig struct {
	Service   config.Component
	EnvParams config.EnvArgs
}

type Plugin func(PluginConfig) []delegatableProvisioner

var Plugins = []Plugin{
	func(cfg PluginConfig) []delegatableProvisioner {
		if cfg.Service.Render == nil {
			return nil
		}

		var provisioners []delegatableProvisioner
		provisioners = append(provisioners, newDelegetableProvisioner("render", &cfg.Service.Render.Delegate, &render.Provisioner{
			Config:    *cfg.Service.Render,
			EnvParams: cfg.EnvParams,
		}))
		return provisioners
	},
	func(cfg PluginConfig) []delegatableProvisioner {
		if cfg.Service.KubernetesResources == nil {
			return nil
		}

		var provisioners []delegatableProvisioner
		provisioners = append(provisioners, newDelegetableProvisioner("k8s", cfg.Service.KubernetesResources.Delegate, &builtin.BuiltinKubernetesProvisioner{
			Config: *cfg.Service.KubernetesResources,
		}))
		return provisioners
	},
	func(cfg PluginConfig) []delegatableProvisioner {
		if cfg.Service.AWSResources == nil {
			return nil
		}

		var provisioners []delegatableProvisioner
		provisioners = append(provisioners, newDelegetableProvisioner("aws", cfg.Service.AWSResources.GitOps, &builtin.BuiltinAWSProvisioner{
			Config: *cfg.Service.AWSResources,
		}))

		return provisioners
	},
	func(cfg PluginConfig) []delegatableProvisioner {
		var provisioners []delegatableProvisioner

		if cfg.Service.ArgoCD.App != nil {
			provisioners = append(provisioners, newDelegetableProvisioner("argocdapp", cfg.Service.ArgoCD.GitOps, &builtin.BuiltinArgoCDAppProvisioner{
				Config:    *cfg.Service.ArgoCD.App,
				EnvParams: cfg.EnvParams,
			}))
		}

		return provisioners
	},
}

func newDelegetableProvisioner(name string, g *config.Delegate, p plugin.Provisioner) delegatableProvisioner {
	return delegatableProvisioner{
		name:        name,
		Delegate:    g,
		Provisioner: p,
	}
}

// delegatableProvisioner is a provisioner that can be triggered by a repository_dispatch event,
// and can be applied by any of the following 4 ways:
// - repository_dispatch events that each delegate any of the following 3 runs
// - git commits to the repos and the branches that contain the gitops configs
// - pull-requests to the repos and the branches that contain the gitops configs
// - Local file changes followed by provider-specific commands (like kubectl-apply and terraform-apply)
//
// The embedded Provisioner does the last one, while the rest is done by the Apply method
// of the indirect provisioner.
type delegatableProvisioner struct {
	name string

	triggeredViaRepositoryDispatch bool

	*config.Delegate

	plugin.Provisioner
}

// prepare prepares the provisioner for the Apply and Destroy methods.
// The passed store.Store that is linked to either a temporary directory or
// the specified directory in the clone of the gitops repository.
func (p *delegatableProvisioner) prepare(ctx context.Context, op string, ds store.Store) (*plugin.RenderResult, error) {
	return ds.Transact(func(path string) (*plugin.RenderResult, error) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getwd: %w", err)
		}

		defer func() {
			if err := os.Chdir(wd); err != nil {
				panic(err)
			}
		}()

		// Render wants the current working directory to be the directory that the provisioner
		// should render the configuration to.
		if err := os.Chdir(path); err != nil {
			return nil, fmt.Errorf("chdir to %s: %w", path, err)
		}

		r, err := p.Provisioner.Render(ctx, ".")
		if err != nil {
			return nil, fmt.Errorf("render: %w", err)
		}

		return r, nil
	})
}

func (p *delegatableProvisioner) Apply(ctx context.Context) (*Result, error) {
	return p.run(ctx, "apply", func(r *plugin.RenderResult) (*plugin.Result, error) {
		return p.Provisioner.Apply(ctx, r)
	})
}

func (p *delegatableProvisioner) Destroy(ctx context.Context) (*Result, error) {
	return p.run(ctx, "destroy", func(_ *plugin.RenderResult) (*plugin.Result, error) {
		return p.Provisioner.Destroy(ctx)
	})
}

func (p *delegatableProvisioner) run(ctx context.Context, op string, fn func(*plugin.RenderResult) (*plugin.Result, error)) (*Result, error) {
	var repositoryDispatches []*config.RepositoryDispatch

	var renderRes *plugin.RenderResult

	if p.Delegate != nil {
		// We have to prevent infinite loop of the repository_dispatch events,
		// and that's why we check if the current run is triggered by a repository_dispatch event.
		if p.Delegate.RepositoryDispatch != nil && !p.triggeredViaRepositoryDispatch {
			repositoryDispatches = append(repositoryDispatches, p.Delegate.RepositoryDispatch)

			return &Result{
				RepositoryDispatches: repositoryDispatches,
			}, nil
		}
	}

	ds := store.Init(p.name, time.Now(), p.Delegate)

	r, err := p.prepare(ctx, op, ds)
	if err != nil {
		return nil, err
	}

	renderRes = r

	if err := ds.Commit(ctx, "automated commit", "n/a"); err != nil {
		return nil, err
	}

	if p.Delegate != nil && (p.Delegate.Git != nil || p.Delegate.PullRequest != nil) {
		return &Result{}, nil
	}

	pluginRes, err := fn(renderRes)
	if err != nil {
		return nil, err
	}

	return &Result{
		Result: *pluginRes,
	}, nil
}
