package config

type GitOps struct {
	// Repo is either REPO/NAME or URL of the git repository that contains the gitops config.
	// A gitops config can be either a directory or a file, that contains Kubernetes manifests,
	// kustomize config, or Terraform workspaces.
	Repo string `yaml:"repo"`

	// Branch is the branch of the git repository that contains the gitops config.
	// It cannot be empty.
	Branch string `yaml:"branch"`

	// Path is the path to the directory or file that contains the gitops config.
	// It cannot be empty.
	Path string `yaml:"path"`

	// PullRequest specifies whether the gitops config is updated via pull request.
	// If false, prenv pushes directly to the branch that contains the gitops config.
	// If true, prenv creates a feature branch, pushes to the feature branch, and creates a pull request.
	// To be clear, the Branch field serves as the base branch of the pull request.
	PullRequest *PullRequest `yaml:"pullRequest"`

	// WorkflowDisatpch specifies whether the gitops config is updated via GitHub workflow_dispatch.
	//
	// If false, prenv pushes directly to the branch that contains the gitops config,
	// optionally creating a pull request depending on the PullRequest field.
	//
	// If true, prenv triggers a GitHub workflow_dispatch event, containing
	// all the information required to update the gitops config.
	// The workflow_dispatch event is sent to the repository specified by Repo,
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
	// - via workflow_dispatch
	// - directly to the branch
	//
	// The workflow_dispatch event is the most flexible way to update the gitops config,
	// because it's actually up to the target repository to decide how to update the gitops config.
	//
	// For convenience, prenv can be run on Actions workflows in both the source and target repositories.
	// prenv run on the source repository is responsible for triggering the workflow_dispatch event.
	// prenv run on the target repository is responsible for updating the gitops config.
	//
	// As the workflow_dispatch inputs contain everything needed to update the gitops config,
	// prenv run on the target repository doesn't need to fetch the source repository.
	//
	// It is also optional to have a prenv.yaml in the target repository,
	// because the workflow_dispatch inputs contain all the information required to update the gitops config.
	WorkflowDispatch *WorkflowDispatch `yaml:"workflowDispatch"`
}

type PullRequest struct{}

type WorkflowDispatch struct{}
