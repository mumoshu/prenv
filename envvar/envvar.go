package envvar

const (
	// Prefix is the prefix of the environment variables used by prenv.
	// All the environment variables used by prenv start with this prefix.
	//
	// However, note that the environment variables can be split into two groups:
	// 1. The environment variables for operatioanl settings
	// 2. The environment variables for the configuration of the pull-request environment
	//
	// The environment variables for operational settings are used by prenv itself,
	// and is not part of the configuration of the pull-request environment.
	//
	// Examples of the environment variables for operational settings are:
	// - GITHUB_TOKEN
	// - GIT_ROOT
	// - COMMIT_AUTHOR_USER_NAME
	// - STATE_FILE_PATH
	// - CONFIGMAP_NAME
	Prefix = "PRENV_"

	//
	// Operational settings
	//

	ConfigMapName           = Prefix + "CONFIGMAP_NAME"
	GitRoot                 = Prefix + "GIT_ROOT"
	GitCommitAuthorUserName = Prefix + "COMMIT_AUTHOR_USER_NAME"
	GitCommitAuthorEmail    = Prefix + "COMMIT_AUTHOR_EMAIL"

	GitHubToken = "GITHUB_TOKEN"

	// StateFilePath is the path to the file that stores the state of the environment.
	//
	// This file is usually stored in either a local git repository or a remote git repository.
	//
	// When EnvVarGitRepoURL is set, this file is stored in the remote git repository.
	// When EnvVarGitRepoURL is not set, this file is stored in the local git repository.
	//
	// If the file is stored in the local git repository, prenv internally deduce the remote repository
	// URL from the local git repository URL, and push the local git repository to the remote repository.
	StateFilePath = Prefix + "STATE_FILE_PATH"

	//
	// Configuration of the pull-request environment
	//
	GitRepoURL = Prefix + "GIT_REPO_URL"
	BaseBranch = Prefix + "BASE_BRANCH"

	// This is used to configure the GitHub API base URL for testing.
	GitHubBaseURL = Prefix + "GITHUB_BASE_URL"

	// This is used to configure the alternative base URL for GitHub's HTTP services.
	// Mainly for swapping out github.com for testing,
	// but also useful for GitHub Enterprise.
	GitHubEnterpriseURL = Prefix + "GITHUB_ENTERPRISE_URL"

	// RawConfig contains the whole content of the prenv.yaml file.
	//
	// It is used to pass the content of the prenv.yaml file from the source repository
	// to the target repository when gitops with workflow_dispatch is used.
	//
	// When this environment variable is set, prenv does not read the prenv.yaml file.
	RawConfig = Prefix + "RAW_CONFIG"

	// GITHUB_EVENT_PATH is the path to the file that contains the event payload.
	// This environment variable is set by GitHub Actions.
	//
	// prenv uses this environment variable to get the event payload.
	// prenv uses the event payload in two ways:
	// 1. To get the pull-request information like the pull-request number
	// 2. To get the content of the prenv.yaml file and the derived configuration when gitops with workflow_dispatch is used
	//
	// 1. is used when prenv is invoked by the pull_request event.
	// 2. is used when prenv is invoked on the target repository by the workflow_dispatch event.
	//    The workflow_dispatch is usually triggered by prenv ran by the pull_request event
	//    on the source repository.
	//
	// https://docs.github.com/en/actions/reference/environment-variables#default-environment-variables
	GitHubEventPath = "GITHUB_EVENT_PATH"

	// https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
	GitHubRepository = "GITHUB_REPOSITORY"
)
