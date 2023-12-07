package state

type State struct {
	EnvironmentNames []string `yaml:"environmentNames"`
}

func (s *State) AddEnvironmentName(envName string) {
	s.EnvironmentNames = append(s.EnvironmentNames, envName)
}

func (s *State) DeleteEnvironmentName(envName string) {
	var envs []string
	for _, name := range s.EnvironmentNames {
		if name != envName {
			envs = append(envs, name)
		}
	}
	s.EnvironmentNames = envs
}
