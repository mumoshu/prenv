These scripts are used to manually test the prenv tooling.

Pre-requisites:

- awscli. Install it with the instructions [here](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html).

## Manual testing

We use the following conditions in the tests:

- `prenv-apps` is the namespace where the ArgoCD applications for prenvs are created.
- `prenv` is the namespace where the prenv components (sqs-forwarder, outgoing-webhook) are deployed.

We don't currently care about:

- Where the ArgoCD server is deployed because prenv doesn't interact with it directly.

If you are going to run the tests in an EKS cluster that already has ArgoCD installed, you might
want to follow these steps:

- Ensure that you have run `aws eks update-kubeconfig --name <cluster-name>` and that you are
  using the correct context.
- Ensure that you have access, in terms of AWS IAM, to create and delete SQS queues.
- Update [`prenv.yaml`](prenv.yaml) with the your own settings:
  - `awsResources`
    - If you have an existingi SQS queue to be reused, you might want do update the settings to something lilke this:
      ```yaml
      awsResources:
        sourceQueueCreate: true
        sourceQueueURL: yourpreferredname
        # By setting this to false, prenv will not create the destination queue and instead will use the existing one
        destinationQueueCreate: false
        destinationQueueURL: <yourexistingqueue name or url>
        # Keep this to true for letting prenv to create the queues for prenvs
        destinationQueuesCreate: true
      ```
  - `environment.argocdApp`
    - `repoURL`: Git repository
    - `path`: The path to the manifests
    - `image`: The image to use for the application
  directory you want prenv to use.
- Run `02-prenv-init`
- Run `03-prenv-apply`
- Somehow see if the app is working as expected.
- Run `04-prenv-destroy`
- Run `05-prenv-deinit`
