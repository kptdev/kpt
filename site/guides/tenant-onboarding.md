# Tenant onboarding use-case

We have seen that in large organizations using kubernetes, there is a platform
team (or infrastructure team) that is responsible for managing the kubernetes
clusters. Typically a kubernetes cluster is shared by multiple teams to run
different types of workloads. One of the common use-cases platform teams have
is onboarding a new tenant on the kubernetes cluster. In this guide, you will
learn - how you can use kpt to address the tenant use-case. Though this guide
focuses on the tenant use-case, the pattern for package workflow discussed here
can be applied to other use cases as well.

**Note:** This guide is inspired by the [kube-common-setup](https://github.com/nghnam/kube-common-setup)
helm chart.

## Terminology

Before we jump into the tutorial, let’s go over some terminologies that are 
used throughout this guide.

### Tenant

A tenant represents a collection of related infra/app resources and needs to be
isolated from others resources, for example,a microservice, a workload, a team
(group of developers) sharing common infra resources.

### Platform team

Central infra team responsible for managing the Kubernetes cluster and is
typically have higher (administrative) privileges on the Kubernetes cluster

### App developer team

The team responsible for operating the tenant. Once the tenant is provisioned,
this team will typically deploy resources (workload, services etc) in the tenant.

## Package Organization

There are many ways to organize the tenant package and its variants. In this guide,
we will explore one of the patterns where we keep the packages and their variants
in different repos as shown in the figure below.

TODO(droot): export the drawing from Google docs and insert here.

![drawing](https://docs.google.com/drawings/d/12345/export/png)

Package catalog repo contains kpt packages that will be used to create variants
of the packages. Platform repo contains the variants of the packages and there
is one-to-many relationship between packages in catalog to platform repo. This
organization has many benefits such as:

* It is easy to discover the packages and its variants.
* It makes it easy to enforce different constraints/invariants on package and
  package variants.
* It allows flexibility in roles/permissions for package publishing/consumption.
* Platform repo also serves as the deployment repository.

Example of [Pkg Catalog Repo](https://github.com/droot/pkg-catalog) and
[Platform Repo](https://github.com/droot/platform).

## Tenant package

The tenant package should have a good set of defaults configuration so that it
can work as a good starting point for most of the tenants. Application developer
teams can customize the tenant package over time as their need evolves.

The tenant package should include invariants (guardrails) that prevent
mis-configuration. For example, each tenant package shouldn’t have more than
one namespace, resource quota shouldn’t exceed limits etc.

One of the key principles to keep in mind is that the tenant package shouldn’t
try to offer all the possible customization options. The tenant package should
offer a reasonable set of defaults with required constraints. Downstream users
of the tenant package can directly edit, add/delete resources as per their needs.

Here is an example of a [basic tenant package](https://github.com/droot/pkg-catalog). 

```shell

pkg-catalog $ kpt pkg tree tenant
Package "tenant"
├── [Kptfile]  Kptfile tenant
├── [namespace.yaml]  Namespace tenant-name
├── [ns-invariant.yaml]  StarlarkRun ns-invariant
├── [quota.yaml]  ResourceQuota tenant-name/quota
├── [role-binding.yaml]  RoleBinding tenant-name/sa-admin
└── [service-account.yaml]  ServiceAccount tenant-name/sa

```

The tenant package’s Kptfile offers basic customization and enforces single
namespace constraint.

```shell
pkg-catalog $ cat tenant/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: tenant
info:
  description: Base tenant package
pipeline:
  mutators:
    - image: set-namespace:v0.1
      configMap:
        namespace: tenant-name # ←- will be customized for pkg variant
  validators:
    - image: gcr.io/kpt-fn/starlark:v0.3
      configPath: ns-invariant.yaml

```

### Optional resources

Sometimes customizing a package may require adding a new resource. For example,
if a team wants to define rules for isolating the network for a tenant, they
need to define network policy resources. Our tenant package has a hidden
directory called `.snippets` that contains snippets of such resources that a
package consumer may need. Platform team can also surface canned configuration
for example quota.medium.yaml can be used for medium size workloads.

Also `.krmignore` files instructs kpt to ignore all resources in the `.snippets`
directory for other workflows.

```shell
pkg-catalog $ ls tenant/.snippets/
isolate-network-policy.yaml  quota.large.yaml  quota.medium.yaml
quota.small.yaml  role-binding.yaml  service-account.yaml

pkg-catalog $ cat tenant/.krmignore
.snippets

```

### Publishing Tenant Package

So once you are happy with the tenant package, you can publish the tenant
package by tagging the version as shown below:

```shell
# Assuming you are in the pkg catalog repo where tenant package exists

# Pl. remove the inventory information from the Kptfile before publishing

# create new tag
$ git tag tenant/v0.1 main

# push the tag to the upstream
$ git push origin tenant/v0.1

```

## Tenant onboarding workflow

Now, let’s take a look at how tenant onboarding will work. Steps described below
can be done by a member of the platform team or application developer team. The
good thing is that it enables self-service workflow for application teams where
they can request for new tenants by simply issuing a PR against the platform repo.

```shell

# assuming you have already a fork of the platform repo
# (remote origin pointing to the fork).

# create a new branch to onboard new tenant
$ git checkout -b onboarding-tenant-a

$ cd tenants
$ kpt pkg get <pkg-catalog-repo>/tenant@v0.1 tenant-a

# tenant customizations:
# change the namespace to tenant-a in the tenant-a/Kptfile
# configure the quota.yaml if needed or
# or cp .snippets/quota.{small, large, medium}.yaml as quota.yaml
# cp .snippets/isolate-network.yaml if isolation is needed
# or directly edit the resources

$ kpt fn render tenant-a

# if all invariants passed, then we are all set.

$ git commit -am "added tenant-a"
$ git push origin onboarding-tenant-a

# make a pull request for platform team to merge
# TODO (link to an example PR will be great here)
```

## Day 2 use-cases

Platform team can evolve the tenant package over time for example, introducing
additional invariants to be enforced, updating the defaults for quota etc. So
assuming a new version of the tenant package `tenant/v0.2` has been published.
Let’s go through the steps needed to update the tenant package.

```shell

# assuming you have already a fork of the platform repo
# (remote origin pointing to the fork).

# create a new branch to update tenant
$ git checkout -b update-tenant-a

$ kpt pkg update tenant-a@tenants/v0.2

$ kpt fn render tenant-a

# if all invariants passed, then we are all set.

$ git commit -am "updated tenant-a to newer version"
$ git push origin update-tenant-a

# make a pull request for platform team to merge
# TODO (link to an example PR will be great here)

// TODO: explain the edge cases where `pkg update` may not be smooth.

```

Note that the platform team can add/update the snippets in the tenant package.
`kpt pkg update` updates the hidden snippets directory automatically but
resources created by copying snippets will not be updated automatically and
need to be updated manually if needed.

## Summary

So, in this guide, how platform teams can enable self service workflow for
application teams to onboard a new tenant. In the next guide, we will explore
how platform teams can do it at a scale when there are hundreds of tenants
provisioned on a kubernetes cluster. Next guides will explore package lifecycle
(pkg diff/update) use cases at large scale.

## Friction Encountered

* `set-namespace` function doesn’t update the namespace for service account in
 subjects in rolebinding
* `kpt pkg update` doesn’t merge pipelines from upstream packages smoothly.
  Was not able to find the fully qualified version for kpt functions.
* `kpt fn render` image-pull-policy should be `ifNotPresent` by default. It
  slows the `fn render` unnecessarily.

## Quick links

[Pkg Catalog Repo](https://github.com/droot/pkg-catalog)
[Platform Repo](https://github.com/droot/platform)
[kube-common-setup helm chart](https://github.com/nghnam/kube-common-setup)
