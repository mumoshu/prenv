package config

import (
	"gopkg.in/yaml.v2"
)

// Config defines the configuration for prenv.
// It is used for declaring the desired state of the pull-request environments.
//
// This includes both the configuration read from the prenv.yaml file,
// and the non-operational settings passed via environment variables.
//
// The configuration does not contain any operational settings.
// Operational settings are passed to prenv via environment variables,
// and handled outside of this configuration.
//
// See envvar/envvar.go for the list of the environment variables used by prenv for operational settings.
type Config struct {
	// EnvironmentNameTemplate is the Go template used to generate the name of the environment
	// It is `{{ .Name }}-{{ .PullRequestNumber }}` by default,
	// where the Name is the name of the ArgoCD application and the PullRequestNumber is the number of the pull request.
	// Name corresponds to Environment.ArgoCDApp.Name.
	EnvironmentNameTemplate string `yaml:"nameTemplate,omitempty"`

	NamePrefix string `yaml:"namePrefix,omitempty"`

	// Shared is the shared service that is shared by all the pull request environments.
	Shared *Component `yaml:"shared,omitempty"`

	// Dedicated is the service that is deployed to the Per-Pull Request Environment.
	Dedicated *Component `yaml:"dedicated,omitempty"`

	// EnvArgs is the set of arguments to be passed to the environment generator.
	// This is populated when prenv is firstly invoked by GitHub Actions,
	// and propagated to delegated prenv runs.
	// In turn, EnvArgs is used to update the shared stack and create/update/destroy the
	// dedicated stack.
	EnvArgs *EnvArgs `yaml:"args,omitempty"`
}

func (c Config) DeepCopy() Config {
	data, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}

	var c2 Config
	if err := yaml.Unmarshal(data, &c2); err != nil {
		panic(err)
	}

	return c2
}
