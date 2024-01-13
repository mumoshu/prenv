package store

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/mumoshu/prenv/provisioner/plugin"
)

// Local is a store implementation that writes to the local filesystem.
type Local struct {
	fs billy.Filesystem
}

func newLocal(id string) *Local {
	pwd, _ := os.Getwd()
	dotPrenv := filepath.Join(pwd, ".prenv")
	if _, err := os.Stat(dotPrenv); os.IsNotExist(err) {
		if err := os.MkdirAll(dotPrenv, 0755); err != nil {
			panic(err)
		}
	}
	dir := filepath.Join(dotPrenv, id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(err)
		}
	}

	fs := osfs.New(dir)

	return &Local{
		fs: fs,
	}
}
func (f *Local) Transact(fn func(path string) (*plugin.RenderResult, error)) (*plugin.RenderResult, error) {
	r, err := fn(f.fs.Root())
	return r, err
}

func (f *Local) Put(ctx context.Context, path, content string) error {
	dir, name := filepath.Split(path)
	return f.Write(dir, File{Name: name, Content: content})
}

func (f *Local) List(ctx context.Context, dir string) ([]string, error) {
	return nil, nil
}

func (f *Local) Get(ctx context.Context, dir string) (*string, error) {
	return nil, nil
}

func (f *Local) Delete(ctx context.Context, dir string) error {
	return nil
}

func (f *Local) Commit(ctx context.Context, subject, body string) error {
	return nil
}

type File struct {
	Name    string
	Content string
}

func (f *Local) Write(dir string, files ...File) error {
	for _, file := range files {
		if err := f.fs.MkdirAll(dir, 0755); err != nil {
			return err
		}

		p := f.fs.Join(dir, file.Name)

		f, err := f.fs.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		defer f.Close()

		if _, err := f.Write([]byte(file.Content)); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}
	}

	return nil
}
