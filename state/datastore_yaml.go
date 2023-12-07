package state

import (
	"context"

	yaml "github.com/goccy/go-yaml"
)

type yamlDataStore struct {
	Data []byte
}

var _ datastore = &yamlDataStore{}

func (s *yamlDataStore) getState(ctx context.Context) (*State, error) {
	state := &State{}

	if err := yaml.Unmarshal(s.Data, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (s *yamlDataStore) setState(ctx context.Context, state *State) error {
	yamlData, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	s.Data = yamlData

	return nil
}

func (s *yamlDataStore) getData() []byte {
	return s.Data
}

func (s *yamlDataStore) load(ctx context.Context, data []byte) (*State, error) {
	s.Data = data

	return s.getState(ctx)
}
