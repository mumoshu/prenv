package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestConfigYAML(t *testing.T) {
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

	t.Run("with shared", func(t *testing.T) {
		r := Config{
			Shared: &Component{
				Render: &Render{
					Files: []RenderedFile{
						{
							Name:            "foo",
							ContentTemplate: "bar",
						},
					},
				},
			},
		}

		got, err := yaml.Marshal(r)
		require.NoError(t, err)

		require.Equal(t, `shared:
  render:
    files:
    - name: foo
      contentTemplate: bar
`, string(got))

		var rev Config

		err = yaml.Unmarshal(got, &rev)
		require.NoError(t, err)

		require.Equal(t, r, rev)
	})

	t.Run("with dedicated", func(t *testing.T) {
		r := Config{
			Dedicated: &Component{
				Render: &Render{
					Files: []RenderedFile{
						{
							Name:            "foo",
							ContentTemplate: "bar",
						},
					},
				},
			},
		}

		got, err := yaml.Marshal(r)
		require.NoError(t, err)

		require.Equal(t, `dedicated:
  render:
    files:
    - name: foo
      contentTemplate: bar
`, string(got))

		var rev Config

		err = yaml.Unmarshal(got, &rev)
		require.NoError(t, err)

		require.Equal(t, r, rev)
	})

	t.Run("with dedicated components", func(t *testing.T) {
		r := Config{
			Dedicated: &Component{
				Components: map[string]Component{
					"svc1": {
						Render: &Render{
							Files: []RenderedFile{
								{
									Name:            "foo",
									ContentTemplate: "bar",
								},
							},
						},
					},
				},
			},
		}

		got, err := yaml.Marshal(r)
		require.NoError(t, err)

		require.Equal(t, `dedicated:
  components:
    svc1:
      render:
        files:
        - name: foo
          contentTemplate: bar
`, string(got))

		var rev Config

		err = yaml.Unmarshal(got, &rev)
		require.NoError(t, err)

		require.Equal(t, r, rev)
	})
}
