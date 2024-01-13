# prenv

**prenv** is a toolkit for creating and managing Per-Pull Request Environments for your projects.

By saying a "toolkit", it means that it is a set of a CLI app and long-running apps that together gives you a handy way to manage PR envs.

The CLI app is supposed to run locally and on CI, where the long-running apps are supposed to run locally(for testing) and remotely(for production).

**prenv** is currently composed of the following tools:

- **prenv**: A CLI tool for creating and managing Per-Pull Request Environments.
- **prenv-sqs-forwarder**: A Go application that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
- **prenv-outgoing-webhook**: A Go application that receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.

## Status

**prenv** is currently in alpha. It is not recommended for production use.

## Installation

- Binaries are available via [GitHub Releases](/releases)
- Container images are [available on Docker Hub](https://hub.docker.com/r/mumoshu/prenv)

## Usage

- Create `prenv.yaml`. See [Configuration](#configuration) for the syntax.

- For each PR:
  - Run [prenv-apply](#prenv-apply) to deploy everything needed for a PR env.
  - Do manual testing by interacting the PR env
  - Run [prenv-destroy](#prenv-destroy) to destroy the PR env

## Configuration

`prenv.yaml` in the root of your repository is the configuration file for prenv. It describes how a Per-Pull Request Environment is provisioned.

At the high-level, a configuration is composed of `shared` and/or `dedicated` `components`.

Components can depend on each other. `prenv` follows the dependency graph and deploys components in an order so that the dependencies are satisfied then a component is deployed.

```yaml
shared:
  components:
    mymiddlewares:
      # ...

dedicated:
  components:
    myapi:
      # ...
    myweb:
      needs: ["myapi"]
      # ...
```

Each component is provisioned via a bespoke `provisioner`.

We have the following provisioners for your choice today:

- `render`: renders file(s) from template(s)

Every provisioner supports gitops, pull-request-ops and indirections via GitHub Actions repository dispatches, which means, you can leverage your existing manual and automated workflows to power pull-request environments.

### render provisioner

`render` provisioner renders file(s) from template(s).

A `render` provisioner config that deploys your `myapi` component via terraform gitops would look like the below:

```yaml
dedicated:
  components:
    myapi:
      render:
        git:
          repo: examplegithuborg/yourrepo
          branch: main
          path: path/to/dir/within/yourrepo
          # Does git-add, git-commit, and git-push after rendering files
          push: true
        files:
        # Updates path/to/dir/within/yourrepo/terraform/test.auto.tfvars.json with the dynamic content
        - name: terraform/test.auto.tfvars.json
          contentTemplate: |
            {"prenv_pull_request_numbers": {{ .PullRequestNumbers | toJson }}}
```

See that `git.branch` points to `main`, which means that `prenv` would git-push to the `main` branch directly.

If you'd like human approvals beforehand and you don't like it directly pushing commits, you can just enable the pull-request support by adding `pullRequest: {}`. By adding it, `prenv` commits to a feature branch and submit a pull request against `main`, instead of pushing commits directly to `main`.

```yaml
render:
  git:
    repo: examplegithuborg/yourrepo
    branch: main
    path: path/to/dir/within/yourrepo
    # Does git-add, git-commit, and git-push after rendering files
    push: true
  # ADDED
  pullRequest: {}
  files:
  # Updates path/to/dir/within/yourrepo/terraform/test.auto.tfvars.json with the dynamic content
  - name: terraform/test.auto.tfvars.json
    contentTemplate: |
      {"prenv_pull_request_numbers": {{ .PullRequestNumbers | toJson }}}
```

Oftentimes you have an application repository and a gitops config repository, where you want to trigger a pull-request-environment deployment from the app repository. The deployment runs on the gitops config repositrory. `prenv` supports this use-case via `repositoryDispatch`.

If you want to do the git update to `examplegithuborg/yourrepo` "from within" that repo, just specify the same repository under the `git` and `repositoryDispatch` fields:

```yaml
render:
  repositoryDispatch:
    owner: examplegithuborg
    repo: yourrepo
  git:
    repo: examplegithuborg/yourrepo
    branch: main
    path: path/to/dir/within/yourrepo
    # Does git-add, git-commit, and git-push after rendering files
    push: true
  files:
    # ...
```

## Commands

Run on GitHub Actions Pull Request event:

- [prenv-apply](#prenv-apply) creates a Per-Pull Request Environment.
- [prenv-destroy](#prenv-destroy) deletes a Per-Pull Request Environment.

Run on cluster:

- [prenv-sqs-forwarder](#prenv-sqs-forwarder) forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
- [prenv-outgoing-webhook](#prenv-outgoing-webhook) receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.

### prenv-apply

`prenv-apply` deploys your application to the Per-Pull Request Environment.

A "Pull-request environment" (PR env in short), is a set of components including AWS resources Kubernetes resources that is dedicated to a pull request.

The `apply` command reads some configuration variables from somewhere and creates or updates AWS and Kubernetes resources.

It reads `event.json` that contains the GitHub Actions event payload or alternatively some environment variables that are available to Actions workflows as input.

`prenv-apply` reads the `GITHUB_REF` enviroment variable to extract the pull request number, and creates the Per-Pull Request Environment.

To create the Per-Pull Request Environment, `prenv-apply` finds the `prenv.yaml` file in the root of your repository and runs Terraform to create the environment.

As a marker, `prenv-apply` creates a `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.

`prenv-apply` run is idempotent. It does nothing when there is already a `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.

### prenv-destroy

`prenv-destroy` undeploys your application from the Per-Pull Request Environment.

`prenv-destroy` reads the `GITHUB_REF` enviroment variable to extract the pull request number, and deletes the Per-Pull Request Environment associated with the pull request.

Once the Per-Pull Request Environment is deleted, `prenv-destroy` deletes the `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.

`prenv-destroy` run is idempotent:

- It does nothing when there is no `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.
- In case it failed after terraform-destroy and before deleting the configmap, you can run `prenv-destroy` again to delete the configmap.

### prenv-sqs-forwrder

**usage(note that you can specify multiple downstream queues)**: `prenv-sqs-forwarder -region <region> -queue <queue> -downstream-queue <downstream-queue> -downstream-queue <downstream-queue>`

See [scripts/sqs-forwarder](/scripts/sqs-forwarder) for the example command that uses all the available flags.

In practice, `prenv-init` generates those flags and deploy `sqs-forwarder` onto your Kubernetes cluster so you don't have to.

It might still be useful to know about what are configurable and not and how it works by reading the example command, though.

### prennv-outgoing-webhook

By specifying the base host name, the subdomain will be treated as the environment name to be included in the notification.

It does also read the `X-Prenv-Environment` header and `Host` header to determine the environment name.

**usage**: `prenv-outgoing-webhook -slack-webhook-url <slack-webhook-url> -base-host <base-host>`

See [scripts/outgoing-webhook](/scripts/outgoing-webhook) for the example command that uses all the available flags.

In practice, `prenv-init` generates those flags and deploy `outgoing-webhook` onto your Kubernetes cluster so you don't have to.

It might still be useful to know about what are configurable and not and how it works by reading the example command, though.

## Implementation

`prenv` is basically a wrapper around various tools that are often used to create and manage Per-Pull Request Environments. The tools are:

- Terraform (and its `terraform` CLI tool)
- ArgoCD (and its `argocd` CLI tool)

Depending on the task, `prenv` might run the CLI tools, or call the REST APIs (if any), use a language-specific SDK (if any), or use a Kubernetes client library (if applicable).

Our choice of the programming language for `prenv` is xx(decide and put the language name here), because:

- It is a compiled language, so we can distribute the binary without requiring the users to install the language runtime.
- It has a rich ecosystem of libraries for interacting with various tools and services.
- It is a statically typed language, so we can catch many errors at compile time.

Specifically, we use the following libraries:

- xx(decide and put the library name here) for interacting with Terraform
- xx(decide and put the library name here) for interacting with ArgoCD
- xx(decide and put the library name here) for interacting with Kubernetes

Here's the list of related libraries and tools that we considered but didn't choose:

- https://github.com/aws/aws-sdk-go
- [argo-cd-apiclient](https://github.com/argoproj/argo-cd/tree/master/pkg/apiclient)
  - Huge transitive deps https://argo-cd.readthedocs.io/en/stable/user-guide/import/
- https://github.com/hashicorp/terraform-exec
- https://github.com/gruntwork-io/terratest
  - too much for our use-case
- https://github.com/beelit94/python-terraform
  - no longer maintained?
- python tftest: https://pypi.org/project/tftest/
- https://github.com/boto/boto3
- https://github.com/kubernetes-client/python
- https://github.com/onlinejudge95/argocd
  - no longer maintained?

## License

MIT License