package config

// Delegate contains the configuration for delegating the deployment to
// another workflow in the same repository, or another workflow in another repository.
//
// prenv can deploy the changes immediately, or delegate the deployment to another workflow or tool.
// This is useful when you want to integrate prenv with an existing deployment workflow.
//
// A deployment can be any of the following. Items marked with * are delegations.
// - (*) Update the gitops config directly
// - (*) Update the gitops config via pull request
// - Update the files locally and runs necessary commands to apply the changes (like kubectl-apply and terraform-apply)
// - (*) Trigger repository_dispatch(events) to another repository, which may in turn do any of the following:
//   - (*) Update the gitops config directly
//   - (*) Update the gitops config via pull request
//   - Update the files locally and runs necessary commands to apply the changes (like kubectl-apply and terraform-apply)
type Delegate struct {
	// Git specifies whether the gitops config is loaded from a git repository.
	Git *Git `yaml:"git,omitempty"`

	// PullRequest specifies whether the gitops config is updated via pull request.
	// If false, prenv pushes directly to the branch that contains the gitops config.
	// If true, prenv creates a feature branch, pushes to the feature branch, and creates a pull request.
	// To be clear, the Branch field serves as the base branch of the pull request.
	PullRequest *PullRequest `yaml:"pullRequest,omitempty"`

	// RepositoryDispatch specifies whether the gitops config is updated via GitHub repository_dispatch.
	//
	// If false, prenv pushes directly to the branch that contains the gitops config,
	// optionally creating a pull request depending on the PullRequest field.
	//
	// If true, prenv triggers a GitHub repository_dispatch event, containing
	// all the information required to update the gitops config.
	// The repository_dispatch event is sent to the repository specified by Repo,
	// along with the infromation below:
	// - the Branch field
	// - the Path field
	// - the PullRequest field
	// - prenv.yaml
	// - PR number
	// - Everything needed to generate inputs required to update the gitops config
	//   (e.g. the content of the head commit, the metadata of the PR, and the templates defined in the configuration)
	//
	// At this point we have three ways to update the gitops config:
	// - via pull request
	// - via repository_dispatch
	// - directly to the branch
	//
	// The repository_dispatch event is the most flexible way to update the gitops config,
	// because it's actually up to the target repository to decide how to update the gitops config.
	//
	// For convenience, prenv can be run on Actions workflows in both the source and target repositories.
	// prenv run on the source repository is responsible for triggering the repository_dispatch event.
	// prenv run on the target repository is responsible for updating the gitops config.
	//
	// As the repository_dispatch inputs contain everything needed to update the gitops config,
	// prenv run on the target repository doesn't need to fetch the source repository.
	//
	// It is also optional to have a prenv.yaml in the target repository,
	// because the repository_dispatch inputs contain all the information required to update the gitops config.
	//
	// When prenv ran on the source repository triggers the repository_dispatch event,
	// it marshals the configuration with a slight modification into a JSON string and
	// sends it as the repository_dispatch inputs.
	//
	// The slight modification is that the RepositoryDispatch field is set to nil,
	// so that the prenv ran on the target repository doesn't trigger the repository_dispatch event again
	// and cause an infinite loop.
	RepositoryDispatch *RepositoryDispatch `yaml:"repositoryDispatch,omitempty"`
}

type Git struct {
	// Repo is either REPO/NAME or URL of the git repository that contains the gitops config.
	// A gitops config can be either a directory or a file, that contains Kubernetes manifests,
	// kustomize config, or Terraform workspaces.
	//
	// This can point to the same repository that the prenv.yaml is in and the pull request is made against,
	// or a different target repository that the repository_dispatch is sent to.
	//
	// Regardless, the gitops config is updated in the repository specified by Repo.
	Repo string `yaml:"repo"`

	// Branch is the branch of the git repository that contains the gitops config.
	// It cannot be empty.
	Branch string `yaml:"branch,omitempty"`

	// Path is the path to the directory or file that contains the gitops config.
	// It cannot be empty.
	Path string `yaml:"path,omitempty"`

	// Push specifies whether the gitops config is updated via git push.
	//
	// If false, prenv just clones the repository, may or may not update the gitops config locally,
	// and runs necessary commands to apply the changes (like kubectl-apply and terraform-apply).
	Push bool `yaml:"push,omitempty"`
}

type PullRequest struct{}

// RepositoryDispatch specifies whether the prenv run is triggered via GitHub repository_dispatch.
type RepositoryDispatch struct {
	// Owner is the owner of the repository that the repository_dispatch is sent to.
	Owner string `yaml:"owner"`
	// Repo is the name of the repository that the repository_dispatch is sent to.
	Repo string `yaml:"repo"`
}
