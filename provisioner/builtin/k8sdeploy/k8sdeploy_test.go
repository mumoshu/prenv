package k8sdeploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/render"
	"github.com/stretchr/testify/require"
)

func TestGenerateManifests(t *testing.T) {
	port := 8080
	config := config.KubernetesApp{
		Name:      "myapp",
		Namespace: "myns",
		Command:   "myapp",
		Image:     "myorg/myapp:dev",
		Args: []string{
			"arg1",
			"--arg2",
			"arg2val",
			"--arg3=arg3val",
		},
		Port: &port,
	}

	testGenerateManifests(t, config)
}

func TestGenerateManifestsNoPort(t *testing.T) {
	config := config.KubernetesApp{
		Name:      "myapp",
		Namespace: "myns",
		Command:   "myapp",
		Image:     "myorg/myapp:dev",
		Args: []string{
			"arg1",
			"--arg2",
			"arg2val",
			"--arg3=arg3val",
		},
	}

	testGenerateManifests(t, config)
}

func TestGenerateManifestsEnv(t *testing.T) {
	config := config.KubernetesApp{
		Name:      "myapp",
		Namespace: "myns",
		Command:   "myapp",
		Image:     "myorg/myapp:dev",
		Env: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}

	testGenerateManifests(t, config)
}

func TestGenerateManifestsSecretEnv(t *testing.T) {
	config := config.KubernetesApp{
		Name:      "myapp",
		Namespace: "myns",
		Command:   "myapp",
		Image:     "myorg/myapp:dev",
		SecretEnv: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}

	testGenerateManifests(t, config)
}

func TestGenerateManifestsEnvWithSecretEnv(t *testing.T) {
	config := config.KubernetesApp{
		Name:      "myapp",
		Namespace: "myns",
		Command:   "myapp",
		Image:     "myorg/myapp:dev",
		Env: map[string]string{
			"FOO": "bar",
		},
		SecretEnv: map[string]string{
			"BAZ": "qux",
		},
	}

	testGenerateManifests(t, config)
}

func testGenerateManifests(t *testing.T, config config.KubernetesApp) {
	t.Helper()

	snapshotName := strings.ToLower(
		strings.ReplaceAll(t.Name(), "/", "_"),
	) + ".yaml"

	tmpl := render.Template{
		Name: config.Name + ".yaml",
		Body: TemplateDeployment,
		Data: &config,
	}

	got, err := render.Execute(tmpl)
	require.NoError(t, err)

	var snapshot string

	snapshotPath := filepath.Join("testdata", snapshotName)
	if os.Getenv("PRENV_TEST_TAKE_SNAPSHOT") != "" {
		snapshot = got[0].Content
		t.Logf("Storing snapshot at %s", snapshotPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(snapshotPath), 0755))
		require.NoError(t, os.WriteFile(snapshotPath, []byte(snapshot), 0644))
	} else {
		want, err := os.ReadFile(snapshotPath)
		require.NoError(t, err, "failed to read snapshot. Run `PRENV_TEST_TAKE_SNAPSHOT=1 go test ./...` to update the snapshot")
		snapshot = string(want)
	}

	want := []render.File{
		{
			Path:    "myapp.yaml",
			Content: snapshot,
		},
	}

	diff := cmp.Diff(want, got)
	require.Empty(t, diff)
}
