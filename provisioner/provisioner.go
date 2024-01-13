// Provisioner is a package that provides a way to provision Kubernetes
// and AWS resources.
// There are two types of provisioners:
// - Kubectl provisioner which writes Kubernetes manifests to a store and then runs kubectl apply
// - Tfvars provisioner which writes a .auto.tfvars file to a store and then runs terraform apply
package provisioner

import (
	"context"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/provisioner/plugin"
	"github.com/mumoshu/prenv/store"
)

type provisioner interface {
	Apply(ctx context.Context) (*Result, error)
	Destroy(ctx context.Context) (*Result, error)
	// Prepare prepares the provisioner for the Apply and Destroy methods.
	// The passed store.Store that is linked to either a temporary directory or
	// the specified directory in the clone of the gitops repository.
	Prepare(ctx context.Context, op string, ds store.Store) error
}

type Result struct {
	// RepositoryDispatches is a list of repository_dispatch events that the provisioner wants to trigger.
	RepositoryDispatches []*config.RepositoryDispatch

	plugin.Result
}
