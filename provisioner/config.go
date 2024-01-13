package provisioner

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/envvar"
	"github.com/mumoshu/prenv/ghactions"
	"gopkg.in/yaml.v2"
)

const (
	ConfigFileName = "prenv.yaml"
)

type Config struct {
	*config.Config

	Action      string
	TriggeredBy []string
}

// GetConfig reads the prenv.yaml file, GitHub Actions and prenv specific environment variables,
// and returns the configuration that contains everything needed to create the pull-request environment.
// It returns an error if it fails to read the configuration.
//
// It reads the prenv.yaml file from the current working directory, if any.
//
// It's also worth noting that it can read the config from workflow_dispatch event inputs.
// This is crucial to support the use case where you want to split prenv runs into two steps:
// - prenv-apply in the source repository
// - prenv-apply in the target repository
func GetConfig() (*Config, error) {
	var (
		r      io.Reader
		cfg    config.Config
		inputs ghactions.Inputs
	)

	v := os.Getenv(envvar.RawConfig)
	if v != "" {
		r = strings.NewReader(v)
	} else if err := ghactions.UnmarshalInputs(&inputs); err == nil {
		if inputs.RawConfig == "" {
			return nil, fmt.Errorf("missing required input in actions workflow_dispatch payload: %s", envvar.RawConfig)
		}
		r = strings.NewReader(inputs.RawConfig)
	} else if err := ghactions.UnmarshalClientPayload(&inputs); err == nil {
		if inputs.RawConfig == "" {
			return nil, fmt.Errorf("missing required input in actions repository_dispatch payload: %s", envvar.RawConfig)
		}
		r = strings.NewReader(inputs.RawConfig)
	} else {
		f, err := os.Open(ConfigFileName)
		if err != nil {
			return nil, fmt.Errorf("unable to open config file %s: %w", ConfigFileName, err)
		}
		defer f.Close()

		r = f
	}

	d := yaml.NewDecoder(r)
	d.SetStrict(true)
	if err := d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode yaml: %w", err)
	}

	action, err := ghactions.GetAction()
	if err != nil {
		return nil, fmt.Errorf("unable to get action: %w", err)
	}

	var c Config

	c.Action = action
	c.Config = &cfg
	c.TriggeredBy = inputs.TriggeredBy

	return &c, nil
}
