package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

const (
	tempDirPattern = "prenvrender"
)

// Template is a template of the rendered file.
type Template struct {
	// Name is the name of the Kubernetes application.
	Name string
	// Body is the Go template to be used to render the file.
	Body string
	// Data is the data to be used to render the file.
	Data interface{}
}

// File is the rendered file to be written to the filesystem.
type File struct {
	Path    string
	Content string
}

// ToTempDir generates Kubernetes manifests for the given Kubernetes application.
// The manifests are written to a temporary directory.
// It's the caller's responsibility to delete the directory.
// This package provides a helper function Cleanup for that purpose.
func ToTempDir(ts ...Template) (*string, error) {
	d, err := CreateTempDir()
	if err != nil {
		return nil, err
	}

	dir := *d

	if _, err := ToDir(dir, ts...); err != nil {
		return nil, err
	}

	return d, nil
}

// ToDir render files from the templates and writes them to the given directory.
// It returns the list of the files written to the directory.
func ToDir(dir string, ts ...Template) ([]string, error) {
	var wrote []string

	for _, t := range ts {
		files, err := Execute(t)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			p := filepath.Join(dir, f.Path)
			d := filepath.Dir(p)

			if err := os.MkdirAll(d, 0755); err != nil {
				return nil, err
			}

			if err := os.WriteFile(p, []byte(f.Content), 0644); err != nil {
				return nil, err
			}

			wrote = append(wrote, f.Path)
		}
	}

	return wrote, nil
}

func Cleanup(dir *string) error {
	if dir == nil {
		return nil
	}

	if *dir == "" {
		return nil
	}

	if !strings.Contains(*dir, tempDirPattern) {
		return nil
	}

	return os.RemoveAll(*dir)
}

// Execute executes the given template and returns the files to be written to the filesystem.
func Execute(t Template) ([]File, error) {
	if t.Name == "" {
		return nil, fmt.Errorf("name must not be empty: config=%v", t)
	}

	if t.Body == "" {
		return nil, fmt.Errorf("body must not be empty: config=%v", t)
	}

	if t.Data == nil {
		return nil, fmt.Errorf("data must not be nil: config=%v", t)
	}

	m := template.New(t.Name)
	m = m.Funcs(template.FuncMap{
		"b64enc": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"split": func(sep, s string) []string {
			return strings.Split(s, sep)
		},
		"toJson": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	})
	m, err := m.Parse(t.Body)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err := m.Execute(&buf, t.Data); err != nil {
		return nil, err
	}

	return []File{
		{
			Path:    t.Name,
			Content: buf.String(),
		},
	}, nil
}
