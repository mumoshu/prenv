package config

// KubernetesApp represents the desired state of the Kubernetes application.
// Each Kubernetes application is a set of Kubernetes resources.
type KubernetesApp struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Replicas  *int              `yaml:"replicas"`
	Command   string            `yaml:"command"`
	Image     string            `yaml:"image"`
	Args      []string          `yaml:"args"`
	Port      *int              `yaml:"port"`
	Env       map[string]string `yaml:"env"`
	SecretEnv map[string]string `yaml:"secretEnv"`
}

func (c *KubernetesApp) Clone() KubernetesApp {
	clone := *c
	if c.Replicas != nil {
		replicas := *c.Replicas
		clone.Replicas = &replicas
	}
	clone.Args = append([]string{}, c.Args...)
	return clone
}
