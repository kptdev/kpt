In the following sections of this chapter you will explore package authoring
using Porch. You will need:

* A GitHub repository for your blueprints. An otherwise empty repository with an
  initial commit works best. The initial commit is required to establish the
  `main` branch.
* A GitHub [Personal Access Token](https://github.com/settings/tokens) with
  the `repo` scope for Porch to authenticate with the repository and allow it
  to create commits in the repository.

A repository is a porch representation of either a git repo or an oci registry.
Package revisions always belong to a single repository. A repository exists in
a Kubernetes namespace and all package revisions in a repo also belong to
the same namespace.

Use the `kpt alpha repo register` command to register your repository with
Porch: The command below uses the repository `deployments.git`.
Your repository name may be different; please update the command with the
correct repository name.

```sh
# Register your Git repository:

GITHUB_USERNAME=<GitHub Username>
GITHUB_TOKEN=<GitHub Personal Access Token>
REPOSITORY_ADDRESS=<Your Repository URL>

$ kpt alpha repo register \
  --namespace default \
  --name deployments \
  --deployment \
  --repo-basic-username=${GITHUB_USERNAME} \
  --repo-basic-password=${GITHUB_TOKEN} \
  ${REPOSITORY_ADDRESS}
```

And register the sample repository we used in the [quickstart](./02-quickstart):

```sh
# Register the sample repository:

kpt alpha repo register --namespace default \
  https://github.com/GoogleContainerTools/kpt-samples.git
```

?> Refer to the [register command reference][register-doc] for usage.

You now have two repositories registered, and your repository is marked as
deployment repository. This indicates that published packages in the repository
are considered deployment-ready.

```sh
# Query repositories registered with Porch:
$ kpt alpha repo get
NAME         TYPE  CONTENT  DEPLOYMENT  READY  ADDRESS
deployments  git   Package  true        True   [Your repository address]
kpt-samples  git   Package              True   https://github.com/GoogleContainerTools/kpt-samples.git
```

?> Refer to the [get command reference][get-doc] for usage.

[register-doc]: /reference/cli/alpha/repo/reg/
[get-doc]: /reference/cli/alpha/repo/get/
