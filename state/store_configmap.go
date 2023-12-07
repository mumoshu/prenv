// Package "state" provides a simple state store
// for the entire prenv application.
// It is used to store the up-to-date list of pull-request environments
// which are then used to create or delete the necessary AWS and Kubernetes resources
// to match the desired state.
// The state store is implemented as a Kubernetes ConfigMap.
// The ConfigMap is updated whenever the state changes.
// The ConfigMap is read whenever the state is needed.
// The ConfigMap is stored in the same Kubernetes namespace as the prenv application.
// The ConfigMap is named "prenv-state".
// The ConfigMap has a key "state" whose value is a YAML string.
package state

import (
	"context"
	"fmt"
	"sync"

	// kubernetes
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	// kubernetes client-go
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// yaml
	yaml "github.com/goccy/go-yaml"
)

const (
	DefaultNamespace = "prenv"
	DefaultName      = "prenv-state"
	DefaultKey       = "state"
)

// ConfigMapStore is a simple state store for the entire prenv application.
type ConfigMapStore struct {
	// The Kubernetes client.
	client *kubernetes.Clientset
	// The Kubernetes namespace.
	Namespace string
	// The name of the ConfigMap.
	Name string
	// The key of the ConfigMap.
	Key string
	// The mutex to synchronize access to the ConfigMap.
	mu sync.Mutex
}

var _ Store = &ConfigMapStore{}

// AddEnvironmentName adds the name of a pull-request environment to the state store.
// It returns an error if it fails to add the name to the state store.
// On each call, it reads and parses the state store, adds the name to the state, and writes the state back to the state store.
// It does retries if it failed due to transient failures.
// It does retry also when multiple AddEnvironmentName calls are racing to update the state store.
// It returns and error on a non-transient failure.
//
// A non-transient failure is one of the following:
//
// - The state exists but is not a valid YAML string.
// - The state exists but is not a valid State struct.
//
// A transient failure is one of the following:
// - The state does not exist.
// - The state was unable to be read.
// - The state was unable to be created.
// - The state was unable to be updated.
func (s *ConfigMapStore) AddEnvironmentName(ctx context.Context, name string) error {
	_, err := s.upsertStateConfigMap(ctx, name)
	return err
}

func (s *ConfigMapStore) DeleteEnvironmentName(ctx context.Context, name string) error {
	_, err := s.deleteEnvNameFromState(ctx, name)
	return err
}

func (s *ConfigMapStore) ListEnvironmentNames(ctx context.Context) ([]string, error) {
	state, err := s.getState(ctx)
	if err != nil {
		return nil, err
	}
	return state.EnvironmentNames, nil
}

func (s *ConfigMapStore) getKey() string {
	if s.Key == "" {
		return DefaultKey
	}
	return s.Key
}

func (s *ConfigMapStore) getName() string {
	if s.Name == "" {
		return DefaultName
	}
	return s.Name
}

func (s *ConfigMapStore) getNamespace() string {
	if s.Namespace == "" {
		return DefaultNamespace
	}
	return s.Namespace
}

// upsertStateConfigMap upserts the state ConfigMap.
// It creates a new ConfigMap if it does not exist.
// If it exists, it reads the state from the ConfigMap, adds the environment name to the state,
// and writes the state back to the ConfigMap.
func (s *ConfigMapStore) upsertStateConfigMap(ctx context.Context, envName string) (*State, error) {
	c, err := s.getClient()
	if err != nil {
		return nil, err
	}

	cm, err := c.CoreV1().ConfigMaps(s.getNamespace()).Get(ctx, s.getName(), metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// The ConfigMap does not exist.
			// Create the ConfigMap.
			state := &State{
				EnvironmentNames: []string{
					envName,
				},
			}
			data, err := yaml.Marshal(state)
			if err != nil {
				return nil, err
			}

			cm, err = c.CoreV1().ConfigMaps(s.getNamespace()).Create(ctx, &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: s.getName(),
				},
				Data: map[string]string{
					s.getKey(): string(data),
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("unable to create configmap: %w", err)
			}
		} else {
			return nil, err
		}
	}

	return s.modifyState(ctx, cm, func(s *State) {
		s.AddEnvironmentName(envName)
	})
}

func (s *ConfigMapStore) deleteEnvNameFromState(ctx context.Context, envName string) (*State, error) {
	c, err := s.getClient()
	if err != nil {
		return nil, err
	}

	cm, err := c.CoreV1().ConfigMaps(s.getNamespace()).Get(ctx, s.getName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return s.modifyState(ctx, cm, func(s *State) {
		s.DeleteEnvironmentName(envName)
	})
}

func (s *ConfigMapStore) getState(ctx context.Context) (*State, error) {
	c, err := s.getClient()
	if err != nil {
		return nil, err
	}

	cm, err := c.CoreV1().ConfigMaps(s.getNamespace()).Get(ctx, s.getName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	state := &State{}

	if err := yaml.Unmarshal([]byte(cm.Data[s.getKey()]), state); err != nil {
		return nil, err
	}

	return state, nil
}

func (s *ConfigMapStore) modifyState(ctx context.Context, cm *v1.ConfigMap, modify func(*State)) (*State, error) {
	state := &State{}

	if err := yaml.Unmarshal([]byte(cm.Data[s.getKey()]), state); err != nil {
		return nil, err
	}

	modify(state)

	data, err := yaml.Marshal(state)
	if err != nil {
		return nil, err
	}

	cm.Data[s.getKey()] = string(data)

	c, err := s.getClient()
	if err != nil {
		return nil, err
	}

	_, err = c.CoreV1().ConfigMaps(s.getNamespace()).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to update configmap: %w", err)
	}

	return state, nil
}

func (s *ConfigMapStore) getClient() (*kubernetes.Clientset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		c, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubernetes client config: %w", err)
		}
		clientConfig := clientcmd.NewDefaultClientConfig(*c, nil)
		config, err := clientConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes client config: %w", err)
		}
		s.client = kubernetes.NewForConfigOrDie(config)
	}
	return s.client, nil
}
