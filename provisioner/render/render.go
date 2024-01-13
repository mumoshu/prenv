package render

import (
	"bytes"
	"context"
	"html/template"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/mumoshu/prenv/render"
)

type Provisioner struct {
	Config    config.Render
	EnvParams config.EnvArgs
}

func (p *Provisioner) Render(ctx context.Context, dir string) (*plugin.RenderResult, error) {
	var ts []render.Template

	for _, r := range p.Config.Files {
		name := r.Name
		if r.NameTemplate != "" {
			tmpl, err := template.New("name").Parse(r.NameTemplate)
			if err != nil {
				return nil, err
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, p.EnvParams); err != nil {
				return nil, err
			}
			name = buf.String()
		}

		t := render.Template{
			Name: name,
			Body: r.ContentTemplate,
			Data: p.EnvParams,
		}

		ts = append(ts, t)
	}

	r, err := render.ToDir(dir, ts...)
	if err != nil {
		return nil, err
	}

	return &plugin.RenderResult{
		AddedOrModifiedFiles: r,
	}, nil
}

func (p *Provisioner) Apply(ctx context.Context, r *plugin.RenderResult) (*plugin.Result, error) {
	return &plugin.Result{}, nil
}

func (p *Provisioner) Destroy(ctx context.Context) (*plugin.Result, error) {
	return &plugin.Result{}, nil
}
