package config

// Deploy represents the desired state of the Kubernetes application.
// Each Kubernetes application is a set of Kubernetes resources.
type Deploy struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Replicas  *int     `yaml:"replicas"`
	Command   string   `yaml:"command"`
	Image     string   `yaml:"image"`
	Args      []string `yaml:"args"`
	Port      *int     `yaml:"port"`
}

func (c *Deploy) Clone() Deploy {
	clone := *c
	if c.Replicas != nil {
		replicas := *c.Replicas
		clone.Replicas = &replicas
	}
	clone.Args = append([]string{}, c.Args...)
	return clone
}
