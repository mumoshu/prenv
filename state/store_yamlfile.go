package state

import (
	"context"
	"os"
)

type YAMLFileStore struct {
	// Path is the path to the YAML file that stores the state of the environment.
	//
	// The YAML file must be a valid YAML file that contains a State struct.
	// Any functions that modify the state of the environment will modify the YAML file.
	//
	// If the YAML file does not exist, the store returns error on
	// any functions that read or modify the state of the environment.
	Path string
}

var _ Store = &YAMLFileStore{}

func (s *YAMLFileStore) AddEnvironmentName(ctx context.Context, name string) error {
	state, err := s.getState(ctx)
	if err != nil {
		return err
	}

	state.AddEnvironmentName(name)

	return s.setState(ctx, state)
}

func (s *YAMLFileStore) DeleteEnvironmentName(ctx context.Context, name string) error {
	state, err := s.getState(ctx)
	if err != nil {
		return err
	}

	state.DeleteEnvironmentName(name)

	return s.setState(ctx, state)
}

func (s *YAMLFileStore) ListEnvironmentNames(ctx context.Context) ([]string, error) {
	state, err := s.getState(ctx)
	if err != nil {
		return nil, err
	}

	return state.EnvironmentNames, nil
}

func (s *YAMLFileStore) getState(ctx context.Context) (*State, error) {
	yamlData, err := os.ReadFile(s.Path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ds := &yamlDataStore{Data: yamlData}
	return ds.getState(ctx)
}

func (s *YAMLFileStore) setState(ctx context.Context, state *State) error {
	ds := &yamlDataStore{}
	if err := ds.setState(ctx, state); err != nil {
		return err
	}

	return os.WriteFile(s.Path, ds.Data, 0644)
}
