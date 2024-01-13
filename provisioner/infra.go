// Package "infra" provides the infrastructure for the application.
//
// There are two main functions: Init() and Deinit().
//
// Init() is called when the infrastructure is initialized.
// Deinit() is called when the infrastructure is deinitialized.
//
// The infrastructure is initialized before the firstapplication starts and deinitialized after all the applications are stopped.
// The infrastructure is initialized and deinitialized only once.
package provisioner

import (
	"context"

	"github.com/mumoshu/prenv/config"
	"github.com/mumoshu/prenv/state"
)

func getDestinationQueueURLs(ctx context.Context, cfg config.AWSResources, store state.Store) ([]string, error) {
	envNames, err := store.ListEnvironmentNames(ctx)
	if err != nil {
		return nil, err
	}

	var destinationQueueURLs []string

	for _, name := range envNames {
		destinationQueueURL := cfg.DeriveQueueURL(name)
		destinationQueueURLs = append(destinationQueueURLs, destinationQueueURL)
	}

	return destinationQueueURLs, nil
}
