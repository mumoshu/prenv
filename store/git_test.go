package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/stretchr/testify/require"
)

func TestGit(t *testing.T) {
	token := os.Getenv("PRENV_GITHUB_TOKEN")
	if token == "" {
		t.Skip("PRENV_GITHUB_TOKEN is not set")
	}

	auth := &http.BasicAuth{
		Username: "prenvbot",
		Password: token,
	}

	t.Run("push", func(t *testing.T) {
		tm := time.Now()

		name := fmt.Sprintf("%s-%s", "test", tm.Format("20060102150405"))
		newBranch := fmt.Sprintf("prenv-%s", name)

		gitRoot := ".prenvtest/push"
		require.NoError(t, os.MkdirAll(gitRoot, 0755))

		t.Cleanup(func() {
			require.NoError(t, os.RemoveAll(gitRoot))
		})

		g := newGit(
			auth,
			"main",
			newBranch,
			"https://github.com/mumoshu/prenv-test.git",
			"test author", "test@example.com",
			gitRoot,
			true,
		)

		r, err := g.Transact(func(dir string) (*plugin.RenderResult, error) {
			p := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				return nil, fmt.Errorf("mkdir error: %w", err)
			}
			if err := os.WriteFile(p, []byte("foo"), 0644); err != nil {
				return nil, fmt.Errorf("write error: %w", err)
			}
			return &plugin.RenderResult{
				AddedOrModifiedFiles: []string{name},
			}, nil
		})
		require.NoError(t, err)

		ctx := context.Background()

		require.NoError(t, g.Commit(ctx, name, "test"))

		require.Equal(t, []string{name}, r.AddedOrModifiedFiles)

		require.NoError(t, os.RemoveAll(gitRoot))

		g2 := newGit(
			auth,
			newBranch,
			"",
			"https://github.com/mumoshu/prenv-test.git",
			"test author", "test@example.com",
			gitRoot,
			false,
		)

		r2, err := g2.Transact(func(dir string) (*plugin.RenderResult, error) {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return nil, fmt.Errorf("read error: %w", err)
			}
			if string(data) != "foo" {
				return nil, fmt.Errorf("unexpected content: %s", string(data))
			}
			return &plugin.RenderResult{}, err
		})
		require.NoError(t, err)
		require.Empty(t, r2.AddedOrModifiedFiles)
	})
}
