package config

type Config struct {
	AWSResources        AWSResources        `yaml:"awsResources"`
	KubernetesResources KubernetesResources `yaml:"kubernetesResources"`

	// EnvironmentNameTemplate is the Go template used to generate the name of the environment
	// It is `{{ .Name }}-{{ .PullRequestNumber }}` by default,
	// where the Name is the name of the ArgoCD application and the PullRequestNumber is the number of the pull request.
	// Name corresponds to Environment.ArgoCDApp.Name.
	EnvironmentNameTemplate string `yaml:"nameTemplate"`
	// Environment is the environment that is deployed per pull request.
	Environment `yaml:",inline"`
}
