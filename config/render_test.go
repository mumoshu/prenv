package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRender(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		r := Render{}

		got, err := yaml.Marshal(r)
		require.NoError(t, err)

		require.Equal(t, "{}\n", string(got))

		var rev Render

		err = yaml.Unmarshal(got, &rev)
		require.NoError(t, err)

		require.Equal(t, r, rev)
	})

	t.Run("with git", func(t *testing.T) {
		r := Render{
			Delegate: Delegate{
				Git: &Git{
					Repo: "mumoshu/prenv",
				},
			},
		}

		got, err := yaml.Marshal(r)
		require.NoError(t, err)

		require.Equal(t, `git:
  repo: mumoshu/prenv
`, string(got))

		var rev Render

		err = yaml.Unmarshal(got, &rev)
		require.NoError(t, err)

		require.Equal(t, r, rev)
	})
}
