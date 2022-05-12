​​In this quickstart you will use Porch to discover configuration packages
in a [sample repository](https://github.com/GoogleContainerTools/kpt-samples).

You will use the kpt CLI - the new `kpt alpha` command sub-groups to interact
with the Package Orchestration service.

## Register the repository

Start by registering the sample repository with Porch. The repository already
contains a [`basens`][basens] package.

```sh
# Register a sample Git repository:
$ kpt alpha repo register --namespace default \
  https://github.com/GoogleContainerTools/kpt-samples.git
```

?> Refer to the [register command reference][register-doc] for usage.

The sample repository is public and Porch therefore doesn't require
authentication to read the repository and discover packages within it.

You can confirm the repository is now registered with Porch by using the
`kpt alpha repo get` command. Similar to `kubectl get`, the command will list
all repositories registered with Porch, or get information about specific ones
if list of names is provided

```sh
# Query repositories registered with Porch:
$ kpt alpha repo get
NAME         TYPE  CONTENT  DEPLOYMENT  READY  ADDRESS
kpt-samples  git   Package              True   https://github.com/GoogleContainerTools/kpt-samples.git
```

?> Refer to the [get command reference][get-doc] for usage.

From the output you can see that:

* the repository was registered by the name `kpt-samples`. This was chosen
  by kpt automatically from the repository URL, but can be overridden
* it is a `git` repository (OCI repositories are also supported, though
  currently with some limitations)
* the repository is *not* a deployment repository. Repository can be marked
  as deployment repository which indicates that packages in the repository are
  intended to be deployed into live state.
* the repository is ready - Porch successfully registered it and discovered
  packages stored in the repository.

The Package Orchestration service is designed to be part of the Kubernetes
ecosystem. The [resources] managed by Porch are KRM resources.

You can use the `-oyaml` to see the YAML representation of the repository
registration resource:

?> kpt uses the same output format flags as `kubectl`. Flags with which you are
already familiar from using `kubectl get` will work with the kpt commands
that get or list Porch resources.

```sh
# View the Repository registration resource as YAML:
$ kpt alpha repo get kpt-samples --namespace default -oyaml
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: kpt-samples
  namespace: default
spec:
  content: Package
  git:
    branch: main
    directory: /
    repo: https://github.com/GoogleContainerTools/kpt-samples.git
    secretRef:
      name: ""
  type: git
status:
  conditions:
  - reason: Ready
    status: "True"
    type: Ready
```

Few additional details are available in the YAML listing:

The name of the `main` branch and a directory. These specify location within
the repository where Porch will be managing packages. Porch also analyzes tags
in the repository to identify all packages (and their specific versions), all
within the directory specified. By default Porch will analyze the whole
repository.

The `secretRef` can contain a name of a Kubernetes [Secret][secret] resource
with authentication credentials for Porch to access the repository.

### kubectl

Thanks to the integration with Kubernetes ecosystem, you can also use `kubectl`
directly to interact with Porch, such as listing repository resources:

```sh
# List registered repositories using kubectl
$ kubectl get repository
NAME          TYPE   CONTENT   DEPLOYMENT   READY   ADDRESS
kpt-samples   git    Package                True    https://github.com/GoogleContainerTools/kpt-samples.git
```

You can use kubectl for _all_ interactions with Porch server if you prefer.
The kpt CLI integration provides a variety of convenience features.

## Discover packages

You can use the `kpt alpha rpkg get` command to list the packages discovered
by Porch across all registered repositories.

```sh
# List package revisions in registered repositories
$ kpt alpha rpkg get
NAME                                                   PACKAGE   REVISION   LATEST   LIFECYCLE   REPOSITORY
kpt-samples-da07e9611f9b99028f761c07a79e3c746d6fc43b   basens    main       false    Published   kpt-samples
kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69   basens    v0         true     Published   kpt-samples
```

?> Refer to the [get command reference][get-doc] for usage.

?> The `r` prefix of the `rpkg` command group stands for `remote`. The commands
in the `kpt alpha rpkg` group interact with packages managed (remotely) by Porch
server. The commands in the `rpkg` group are similar to the `kpt pkg` commands
except that they operate on remote packages managed by Porch server rather than
on a local disk.

The output shows that Porch discovered the `basens` package, and found two
different revisions of it. The `v0` revision (associated with the
[`basens/v0`][basens-v0] tag) and the `main` revision associated with the
[`main` branch][main-branch] in the repository.

The `LIFECYCLE` column indicates the lifecycle stage of the package revision.
The package revisions in the repository are *`Published`* - ready to be used.
Package revision may be also *`Draft`* (the package revision is being authored)
or *`Proposed`* (the author of the package revision proposed that it be
published). We will encounter examples of these 

Porch identifies the latest revision of the package (`LATEST` column).

### View package resources

The `kpt alpha rpkg get` command displays package metadata. To view the
_contents_ of the package revision, use the `kpt alpha rpkg pull` command.

You can use the command to output the resources as a
[`ResourceList`][resourcelist] on standard output, or save them into a local
directory:

```sh
# View contents of the basens/v0 package revision
$ kpt alpha rpkg pull kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69 -ndefault

apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: basens
    annotations:
...
```

Add a name of a local directory on the command line to save the package onto
local disk for inspection or editing.

```sh
# Pull package revision resources, save to local disk into `./basens` directory
$ kpt alpha rpkg pull kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69 ./basens -ndefault

# Explore the package contents
$ find basens

basens
basens/README.md
basens/namespace.yaml
basens/Kptfile
...
```

?> Refer to the [pull command reference][pull-doc] for usage.

## Unregister the repository

When you are done using the repository with Porch, you can unregister it:

```sh
# Unregister the repository
$ kpt alpha repo unregister kpt-samples -ndefault
```

## More resources

To continue learning about Porch, you can review:

* [Porch User Guide](/guides/porch-user-guide)
* [Provisioning Namespaces with the UI](/guides/namespace-provisioning-ui)
* [Porch Design Document][design]

[basens]: https://github.com/GoogleContainerTools/kpt-samples/tree/main/basens
[register-doc]: /reference/cli/alpha/repo/reg/
[get-doc]: /reference/cli/alpha/repo/get/
[pull-doc]: /reference/cli/alpha/rpkg/pull/
[unregister-doc]: /reference/cli/alpha/repo/unreg/
[resources]: /guides/porch-user-guide
[secret]: https://kubernetes.io/docs/concepts/configuration/secret/
[basens-v0]: https://github.com/GoogleContainerTools/kpt-samples/tree/basens/v0
[main-branch]: https://github.com/GoogleContainerTools/kpt-samples/tree/main
[resource-list]: /reference/schema/resource-list/
[design]: https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/07-package-orchestration.md
