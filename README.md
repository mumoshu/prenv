# prenv

**prenv** is a toolkit for creating and managing Per-Pull Request Environments for your projects.

**prenv** is currently composed of the following tools:

- **prenv**: A CLI tool for creating and managing Per-Pull Request Environments.
- **prenv-sqs-forwarder**: A Go application that forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
- **prenv-outgoing-webhook**: A Go application that receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.

**prenv** is currently in alpha. It is not recommended for production use.

## Installation

prenv and its tools are available as Docker images on Docker Hub and GitHub Container Registry. Also, there is a Helm chart for installing prenv and its tools to your Kubernetes cluster. [`prenv-init`](#prenv-init) internally uses the chart to install the tools.

## Configuration

`prenv.yaml` in the root of your repository is the configuration file for prenv. It describes how a Per-Pull Request Environment is created.

```yaml
terraform:
  module: "myinfra"
  vars:
    - name: "foo"
      value: "bar"
    - name: "baz"
      valueTemplate: "prenv-{{ .PullRequestNumber }}"
```

## Commands

- [prenv-init](#prenv-init) sets up the prerequisites for creating and managing Per-Pull Request Environments.
- [prenv-deinit](#prenv-deinit) deletes the prerequisites for creating and managing Per-Pull Request Environments.
- [prenv-apply](#prenv-apply) creates a Per-Pull Request Environment.
- [prenv-destroy](#prenv-destroy) deletes a Per-Pull Request Environment.

### prenv-init

`prenv-init` ensures that the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` are deployed to your Kubernetes cluster.

### prenv-deinit

`prenv-deinit` deletes the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` from your Kubernetes cluster.

### prenv-apply

`prenv-apply` reads the `GITHUB_REF` enviroment variable to extract the pull request number, and creates the Per-Pull Request Environment.

To create the Per-Pull Request Environment, `prenv-apply` finds the `prenv.yaml` file in the root of your repository and runs Terraform to create the environment.

As a marker, `prenv-apply` creates a `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.

`prenv-apply` run is idempotent. It does nothing when there is already a `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.

### prenv-destroy

`prenv-destroy` reads the `GITHUB_REF` enviroment variable to extract the pull request number, and deletes the Per-Pull Request Environment associated with the pull request.

Once the Per-Pull Request Environment is deleted, `prenv-destroy` deletes the `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.

`prenv-destroy` run is idempotent:

- It does nothing when there is no `prenv-${PR_NUMBER}` configmap in the namespace of your Kubernetes cluster.
- In case it failed after terraform-destroy and before deleting the configmap, you can run `prenv-destroy` again to delete the configmap.

## prenv-sqs-forwrder

**usage(note that you can specify multiple downstream queues)**: `prenv-sqs-forwarder -region <region> -queue <queue> -downstream-queue <downstream-queue> -downstream-queue <downstream-queue>`

## prennv-outgoing-webhook

By specifying the base host name, the subdomain will be treated as the environment name to be included in the notification.

It does also read the `X-Prenv-Environment` header and `Host` header to determine the environment name.

**usage**: `prenv-outgoing-webhook -slack-webhook-url <slack-webhook-url> -base-host <base-host>`
