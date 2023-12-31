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

- Run [`prenv-init`](#prenv-init) to deploy all the prerequisites onto your Kubernetes cluster.

- SQS-only: [Redeploy existing non-PR environment to work with prenv](#reconfiguring-non-PR-environments)

- For each PR:
  - Run [prenv-apply](#prenv-apply) to deploy everything needed for a PR env.
  - Run [prenv-test](#prenv-test) to run the test(s) you defined
  - Do manual testing by interacting the PR env
  - Run [prenv-destroy](#prenv-destroy) to destroy the PR env

## Configuration

`prenv.yaml` in the root of your repository is the configuration file for prenv. It describes how a Per-Pull Request Environment is created.

```yaml
## The following sqs section is asummed by default
## when you specify just `sqs: {}`
sqs:
  queueNameTemplate: "prenv-{{ .PullRequestNumber }}"
  ## The following attributes are optional.
  ## It must set to `true` if you want prenv to create SQS queues.
  ## Set of `false` when you want to create the queues using e.g. Terraform.
  #create: false

## We currently assume the outgoing webhook is always deployed
## to the same namespace as the argocd application.
## So no additional configuration is required.
#outgoingWebhook: {}

## envvars controls the names of the environment variables
## that are passed to the application for the Per-Pull Request
## Environment.
envvars:
  sqsQueueURL:
    name: "MY_CUSTOM_SQS_QUEUE_URL_ENV_VAR_NAME"
  outgoingWebhookURL:
    name: "MY_CUSTOM_OUTGOING_WEBHOOK_URL_ENV_VAR_NAME"

# baseName is the basename of the environment deployed per pull-request.
baseName: "myapp"

# The below is the default Go template used for generating names of per-pull-request environments.
nameTemplate: "{{ .BaseName }}-{{ .PullRequestNumber }}"

# You need to specify either:
# - argocdApp (in case it's a monolith) or
# - services.$SHORT_NAME.argocdApp (in case it's composed of microservices)

services:
  myweb:
    argocdApp:
      # see below
  myapi:
    argocdApp:
      # see below

## The following argocd section is asummed by default
## when you specify just `argocdApp: {}`
argocdApp:
  namespace: prenv-apps
  destinationNamespace: prenv
  destinationServer: https://kubernetes.default.svc
  repoURL: git@github.com:mumoshu/prenv.git
  path: manifests
  targetRevision: HEAD
  image: mumoshu/prenv-example-app
  # The below is the default appTemplate that is used to render
  # the ArgoCD Application manifest.
  # Each Go template variable looks like `{{ .VarName }}` corresponds
  # to the upper-camel-cased versions of fields like `namespace`, `destinationNamespace`, and so on.
  appTemplate: |
    metadata:
      name: "{{ .Name }}"
      namespace: "prenv"
    spec:
      source:
        # .GitHubRepositoryURL corresponds to the $GITHUB_REPOSITORY
        # environment variable
        repoURL: "{{ .GitHubRepositoryURL }}"
        # .SHA corresponds to the $GITHUB_SHA environment variable
        # available on GitHub Actions.
        targetRevision: "{{ .SHA }}"
        path: "deploy/argocd"
      destination:
        namespace: "prenv-{{ .PullRequestNumber }}"
        server: "https://kubernetes.default.svc"
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
      ## The queue URL is passed to the application via the
      ## environment variable `SQS_QUEUE_URL`.
      env:
      # The env name is configurable via the `envvars.sqsQueueURL` field.
      - name: "SQS_QUEUE_URL"
        value: "{{ .SQSQueueURL }}"
      # The env name is configurable via the `envvars.outgoingWebhookURL` field.
        - name: "OUTGOING_WEBHOOK_URL"
          value: "{{ .OutgoingWebhookURL }}"

terraform:
  module: "myinfra"
  vars:
    - name: "foo"
      value: "bar"
    - name: "baz"
      valueTemplate: "prenv-{{ .PullRequestNumber }}"
    ## In case you want to deploy the app using Terraform AND
    ## the queue is created by prenv-init, you need to pass the
    ## queue name to the Terraform module.
    #- name: "queue_name"
    #  valueTemplate: "prenv-{{ .PullRequestNumber }}"
```

## Reconfiguring non-PR environments

Reconfigure either the sender or the receiver of the original SQS queue to use the SQS managed by `prenv`.

Let's say your application (SQS consumer) takes SQS messages as inputs and your existing non-PR environments looked like the below.

```
SQS publisher --> existing queue ---> SQS consumer
```

To avoid manually replicating SQS messages to trigger SQS consumer across environments for testing, we want to replicate messages sent to `SQS consumer`. That's where `prenv-sqs-forwarder` comes in.

Introducing `prenv`, we want the environments to look like the either of the belows.

Option 1: Reconfigure SQS publisher

```
SQS publisher --> source queue (new) -->
  prenv-sqs-forwarder --> destination queue (existing) --> SQS consumer
                      --> destination queue PR #123    --> PR #123 SQS consumer
                      --> destination queue PR #234    --> PR #234 SQS consumer
```

Option 2: Reconfigure SQS consumer

```
SQS publisher --> source queue (existing) -->
  prenv-sqs-forwarder --> destination queue         --> SQS consumer
                      --> destination queue PR #123 --> PR #123 SQS consumer
                      --> destination queue PR #234 --> PR #234 SQS consumer
```

For Option 1, your `prenv.yaml` would look like:

```
awsResources:
  # We let prenv create the static queue that SQS publisher sends to
  # The value `true` here corresponds to `(new)` of the `source queue (new)` in Option 1.
  sourceQueueCreate: true
  sourceQueueURL: prenv-static-queue
  # We reuse the existing queue here
  # The value `false` corresponds to `(existing)` of the `destination queue (existing)` in Option 1.
  destinationQueueCreate: false
  # However queues for PR envs are created by prenv
  # This corresponds to `destination queue PR #<PR NUMBER>` in the figure above.
  destinationQueuesCreate: true
  destinationQueueURL: testdestinationqueue
```

For Option 2, it would look like:

```
awsResources:
  # We reuse the existing queue here
  # The value `true` here corresponds to `(existing)` of the `source queue (existing)` in Option 2.
  sourceQueueCreate: false
  sourceQueueURL: $URL_OR_NAME_OF_EXISTING_QUEUE
  # We let prenv create the static queue that SQS consumer subscribes to
  # The value `true` corresponds to `(new)` of the `destination queue (new)` in Option 1.
  destinationQueueCreate: true
  # Queues for PR envs are created by prenv anyway
  # This corresponds to `destination queue PR #<PR NUMBER>` in the figure above.
  destinationQueuesCreate: true
  destinationQueueURL: testdestinationqueue
```

## Commands

Run locally or on GitHub Actions:

- [prenv-init](#prenv-init) sets up the prerequisites for creating and managing Per-Pull Request Environments.
- [prenv-deinit](#prenv-deinit) deletes the prerequisites for creating and managing Per-Pull Request Environments.

Run on GitHub Actions Pull Request event:

- [prenv-apply](#prenv-apply) creates a Per-Pull Request Environment.
- [prenv-destroy](#prenv-destroy) deletes a Per-Pull Request Environment.

Run on cluster:

- [prenv-sqs-forwarder](#prenv-sqs-forwarder) forwards messages from an SQS queue to the downstream, Per-Pull Request Environments' SQS queues.
- [prenv-outgoing-webhook](#prenv-outgoing-webhook) receives outgoing webhooks from the Per-Pull Request Environments and forwards them to the Slack channel of your choice.

### prenv-init

`prenv-init` ensures that the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` are deployed to your Kubernetes cluster.

### prenv-deinit

`prenv-deinit` deletes the `prenv-sqs-forwarder` and `prenv-outgoing-webhook` from your Kubernetes cluster.

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