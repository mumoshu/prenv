package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	l := newLocal("test")
	wd, err := os.Getwd()
	require.NoError(t, err)
	var called bool
	l.Transact(func(dir string) (*plugin.RenderResult, error) {
		called = true
		require.Equal(t, filepath.Join(wd, ".prenv/test"), dir)
		return nil, nil
	})
	require.True(t, called)
}
