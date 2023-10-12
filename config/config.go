package config

type Config struct {
	AWSResources        AWSResources        `yaml:"awsResources"`
	KubernetesResources KubernetesResources `yaml:"kubernetesResources"`
	Environment         Environment         `yaml:"environment"`
}
