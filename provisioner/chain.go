package provisioner

import (
	"context"
	"fmt"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/generator"
	"github.com/mumoshu/prenv/ghactions"
	"github.com/mumoshu/prenv/state"
	"gopkg.in/yaml.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// Chain is a provisioner chain that runs multiple provisioners.
//
// It basically consults the config.Config and runs provisioners that are enabled in the config.Config.
// It also runs provisioners that are triggered by the repository_dispatch events only, if any.
type Chain struct {
	// Action is either "prenv-apply" or "prenv-destroy"
	// passed via the repository_dispatch event_type field.
	action string

	cfg config.Config

	provisioners []delegatableProvisioner
}

func ChainFromEnv() (*Chain, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}

	return NewChain(cfg)
}

func NewChain(cfg *Config) (*Chain, error) {
	ctx := context.Background()

	triggeredBy := cfg.TriggeredBy

	var chain Chain

	store := state.NewStore(*cfg.Config)

	envNames, err := store.ListEnvironmentNames(ctx)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("unable to list enviroment names: %w", err)
		}
	}

	if cfg.Shared != nil && cfg.Shared.AWSResources != nil {
		awsResources := *cfg.Shared.AWSResources
		awsResources.DestinationQueueNames = envNames
		cfg.Shared.AWSResources = &awsResources

		destQueueURLs, err := getDestinationQueueURLs(context.Background(), awsResources, store)
		if err != nil {
			return nil, err
		}

		if cfg.Shared.KubernetesResources != nil {
			k8sRes := cfg.Shared.KubernetesResources
			if k8sRes.Image == "" {
				k8sRes.Image = config.DefaultImage
			}
			k8sRes.SQSForwarder.SourceQueueURL = cfg.Shared.AWSResources.GetSourceQueueURL()
			k8sRes.SQSForwarder.DestinationQueueURLs = append(k8sRes.SQSForwarder.DestinationQueueURLs, cfg.Shared.AWSResources.GetDestinationQueueURL())
			k8sRes.SQSForwarder.DestinationQueueURLs = append(k8sRes.SQSForwarder.DestinationQueueURLs, destQueueURLs...)
			cfg.Shared.KubernetesResources = k8sRes
		}
	}

	var envArgs *config.EnvArgs

	if cfg.EnvArgs == nil {
		var err error
		// TODO we might want to trigger this only when the pull-request generator is enabled
		envArgs, err = generator.BuildGitHubActionsPullRequestEnvArgs(*cfg.Config)
		if err != nil {
			return nil, err
		}
	} else {
		envArgs = cfg.EnvArgs
	}

	if err := store.AddEnvironmentName(ctx, envArgs.Name); err != nil {
		return nil, err
	}

	// TODO: Iterate over pull request(s)?

	{
		if envArgs.AppNameTemplate == "" {
			envArgs.AppNameTemplate = "{{ .Environment.Name }}-{{ .Environment.PullRequestNumber }}-{{ .ShortName }}"
		}

		components := map[string]config.Component{}

		if cfg.Shared != nil {
			components[""] = *cfg.Shared
		}

		if cfg.Dedicated != nil {
			p1 := cfg.Dedicated.NamePrefix
			if p1 == "" {
				p1 = "pr-"
			}
			components[p1] = *cfg.Dedicated

			for name, s := range cfg.Dedicated.Components {
				p2 := s.NamePrefix
				if p2 == "" {
					p2 = name + "-"
				}
				components[p1+p2] = s
			}
		}

		for namePrefix, svc := range components {
			for _, p := range Plugins {
				provisioners := p(PluginConfig{
					Service:   svc,
					EnvParams: *envArgs,
				})

				for i := range provisioners {
					provisioners[i].name = namePrefix + provisioners[i].name
				}

				var triggeredProvisioners []delegatableProvisioner

				for i := range provisioners {
					p := provisioners[i]

					if len(triggeredBy) > 0 {
						var found bool
						for _, t := range triggeredBy {
							if t == p.name {
								found = true
								break
							}
						}
						if !found {
							continue
						}

						// This is used to prevent infinite loop of the repository_dispatch events.
						p.triggeredViaRepositoryDispatch = true
					}

					triggeredProvisioners = append(triggeredProvisioners, p)
				}

				chain.provisioners = append(chain.provisioners, triggeredProvisioners...)
			}
		}
	}

	cfg.EnvArgs = envArgs
	chain.cfg = *cfg.Config
	chain.action = cfg.Action

	return &chain, nil
}

type triggeredRepositoryDispatch struct {
	*config.RepositoryDispatch

	// The name of the provisioner that is triggered by the repository_dispatch event.
	// This is used to both prevent infinite loop of the repository_dispatch events,
	// and to identify which provisioner is triggered by the repository_dispatch event.
	provisionerName string
}

type mergedRepositoryDispatch struct {
	*config.RepositoryDispatch

	// The name of the provisioner that is triggered by the repository_dispatch event.
	provisionerNames []string
}

// Apply creates and updates the pull-request environment.
// An apply is idempotent, and can be triggered by either a pull-request event,
// or a workflow_dispatch event.
//
// When it is triggered by a pull-request event, it creates a new pull-request environment
// based on the pull-request number, the content of the head commit, and the configuration
// defined in prenv.yaml.
//
// When it is triggered by a workflow_dispatch event, it creates a new pull-request environment
// based on the workflow_dispatch inputs containing the pull-request number and arbitrary variables.
// The "arbitrary variables" are generated from the metadata of the PR, the content of the head commit,
// and the templates defined in the configuration.
//
// How to create the pull-request environment is defined in the configuration.
// If the configuration contains gitOps fields, it creates the Kubernetes resources and/or
// AWS resources defined in the configuration via GitOps.
// It's up to the gitops repository's CI/CD pipeline to create the Kubernetes resources and/or
// AWS resources in that case.
//
// If the configuration does not contain gitOps field, it creates the Kubernetes resources and/or AWS resources defined in the configuration
// using the built-in provisioners.
func (c *Chain) Apply(ctx context.Context) error {
	_, err := c.run(ctx, ghactions.EventTypeApply, func(ctx context.Context, p delegatableProvisioner) (*Result, error) {
		return p.Apply(ctx)
	})

	return err
}

// Destroy deletes the pull-request environment and reconfigures the infrastructure.
//
// A destroy is idempotent, and can be triggered by either a pull-request event,
// or a workflow_dispatch event.
//
// When it is triggered by a pull-request event, it deletes the pull-request environment
// based on the pull-request number, the content of the head commit, and the configuration
// defined in prenv.yaml.
//
// When it is triggered by a workflow_dispatch event, it deletes the pull-request environment
// based on the workflow_dispatch inputs containing the pull-request number and arbitrary variables.
// The "arbitrary variables" are generated from the metadata of the PR, the content of the head commit,
// and the templates defined in the configuration.
//
// How to destroy the pull-request environment is defined in the configuration.
//
// If the configuration contains gitOps fields, it deletes the Kubernetes resources and/or
// AWS resources defined in the configuration via GitOps.
// It's up to the gitops repository's CI/CD pipeline to delete the Kubernetes resources and/or
// AWS resources in that case.
//
// If the configuration does not contain gitOps field, it deletes the Kubernetes resources and/or AWS resources defined in the configuration
// using the built-in provisioners.
func (c *Chain) Destroy(ctx context.Context) error {
	_, err := c.run(ctx, ghactions.EventTypeDestroy, func(ctx context.Context, p delegatableProvisioner) (*Result, error) {
		return p.Destroy(ctx)
	})

	return err
}

func (c *Chain) Action(ctx context.Context) error {
	switch c.action {
	case ghactions.EventTypeApply:
		return c.Apply(ctx)
	case ghactions.EventTypeDestroy:
		return c.Destroy(ctx)
	}

	return fmt.Errorf("unknown action: %s", c.action)
}

func (c *Chain) run(ctx context.Context, action string, fn func(ctx context.Context, p delegatableProvisioner) (*Result, error)) ([]*mergedRepositoryDispatch, error) {
	var triggeredDispatches []*triggeredRepositoryDispatch
	for _, p := range c.provisioners {
		r, err := fn(ctx, p)
		if err != nil {
			return nil, err
		}

		if len(r.RepositoryDispatches) > 0 {
			for _, d := range r.RepositoryDispatches {
				triggeredDispatches = append(triggeredDispatches, &triggeredRepositoryDispatch{
					RepositoryDispatch: d,
					provisionerName:    p.name,
				})
			}
		}
	}

	var mergedDispatches []*mergedRepositoryDispatch

	for _, d := range triggeredDispatches {
		var found bool
		for _, m := range mergedDispatches {
			if m.RepositoryDispatch == d.RepositoryDispatch {
				m.provisionerNames = append(m.provisionerNames, d.provisionerName)
				found = true
				break
			}
		}
		if !found {
			mergedDispatches = append(mergedDispatches, &mergedRepositoryDispatch{
				RepositoryDispatch: d.RepositoryDispatch,
				provisionerNames:   []string{d.provisionerName},
			})
		}
	}

	// For example, a prenv.yaml portion for kubernetesResources looks like the below when the repository_dispatch is used:
	//
	//	awsResources:
	//	  # awsResources fields follow
	//	kubernetesResources:
	//	  repositoryDispatch:
	//	    owner: mumoshu
	//	    repo: prenv
	//	  # other kubernetesResources fields follow
	//	argocdApp:
	//	  # argocdApp fields follow
	//
	// When prenv run on the source repository triggers the repository_dispatch event on the target repository,
	// it marshals the configuration into a JSON string and
	// sends it as the repository_dispatch inputs:
	//
	//	{
	//	  "raw_config": "<yaml string of the config.Config>",
	//    "triggered_by": ["k8s"]
	//	}
	//
	// So that prenv run on the target repository can use the configuration to deploy the pull-request environment,
	// without triggering the repository_dispatch event again and causing an infinite loop.

	for _, d := range mergedDispatches {
		var inputs ghactions.Inputs

		rawConfig, err := yaml.Marshal(c.cfg)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal config: %w", err)
		}

		inputs.RawConfig = string(rawConfig)
		inputs.TriggeredBy = d.provisionerNames

		if err := ghactions.SendRepositoryDispatch(ctx, action, *d.RepositoryDispatch, inputs); err != nil {
			return nil, fmt.Errorf("unable to send repository_dispatch event: %w", err)
		}
	}

	return mergedDispatches, nil
}
