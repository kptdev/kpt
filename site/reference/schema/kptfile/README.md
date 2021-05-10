# `Kptfile`

```yaml
# API version of the Kptfile.
apiVersion: kpt.dev/v1alpha2
# Always "kptfile."
kind: Kptfile
# Metadata of the resource described by the kptfile.
metadata:
  # Name of this resource.
  name: wordpress
# The upstream locator for a package.
upstream:
  # The type of origin for a package. Only supports "git."
  type: git
  # The locator for a package on Git.
  git:
    # The repository of the package.
    repo: https://github.com/GoogleContainerTools/kpt
    # The sub directory in the git repository.
    directory: /package-examples/wordpress
    # A git branch, tag, or a commit SHA-1.
    ref: v0.1
  # The method used in updating from upstream.
  # Supports "resource-merge," "fast-forward" and "force-delete-replace."
  updateStrategy: resource-merge
# Declares the pipeline of functions used to mutate or validate resources..
pipeline:
  # A list of of KRM functions that mutate resources.
  mutators:
    # Specifies the function container image.
    # Can be fully qualified or infer the host path from
    # kpt's configuration (defaults to gcr.io/kpt-fn)
    - image: gcr.io/kpt-fn/apply-setters:v0.1
    # An inline KRM resource used as the function config.
    # Cannot be specified alongside config or configPath.
    configMap:
    wp-image: wordpress
    wp-tag: 4.8-apache
  # A list of KRM functions that validate resources.
  # Validators are not permitted to mutate resources.
  validators:
      # Specifies the function container image.
      # Can be fully qualified or infer the host path from
      # kpt's configuration (defaults to gcr.io/kpt-fn)
      # The following resolves to gcr.io/kpt-fn/kubeval:v0.1:
    - image: kubeval:v0.1
# Optional information about the package not passed to kpt.
info:
  # The URL of the package's homepage.
  site:
    https://github.com/GoogleContainerTools/kpt
  # The list of emails of the package authors.
  emails:
    - kpt-team@google.com
  # SPDX license identifier. See: https://spdx.org/licenses/
  license: Apache-2.0"
  # The path to documentation about the package.
  doc: README.md
  # A short description of the package.
  description: This is an example wordpress package.
  # A list of keywords for this package.
  keywords:
    - demo package
# Parameters for the inventory object applied to a cluster.
# All of the parameters are required if any are set.
inventory:
  # Name of the inventory resource.
  name: inventory-82393036
  # Namespace for the inventory resource.
  namespace: rbac-error
  # Unique label to identify inventory resource in cluster.
  inventoryID: ed68c3d787d4355ac1886ee852f7a4c0537d9818-1618448890187393263
# The resolved locator for the last fetch of the package.
upstreamLock:
  # The type of origin for a package. Only supports "git."
  type: git
  # The locator for a package on Git.
  git:
    # The repository of the package that was fetched.
    repo: https://github.com/GoogleContainerTools/kpt
    # The sub directory in the git repository that was fetched.
    directory: /package-examples/wordpress
    # The git branch, tag, or a commit SHA-1 that was fetched.
    ref: v0.1
    # SHA-1 for the last fetch of the package.
    commit: e0e0b3642969c2d14fe1d38d9698a73f18aa848f