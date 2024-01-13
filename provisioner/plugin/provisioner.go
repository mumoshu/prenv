package plugin

import (
	"context"
)

type Provisioner interface {
	Apply(ctx context.Context, r *RenderResult) (*Result, error)
	Destroy(ctx context.Context) (*Result, error)
	// Render renders the provisioner's configuration to the directory.
	//
	// The provisioner framework will prepare the directory for the provisioner
	// by cloning the gitops repository and may chdir to the specific directory in the repository,
	// or by creating a temporary directory and may chdir to it.
	//
	// Nevertheless, the dir argument is the directory that the provisioner should render the configuration to.
	Render(ctx context.Context, dir string) (*RenderResult, error)
}

type RenderResult struct {
	AddedOrModifiedFiles []string
	DeletedFiles         []string
}
