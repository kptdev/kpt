# `Kptfile`

```yaml
# API version of the Kptfile.
apiVersion: kpt.dev/v1alpha2

# Always "kptfile."
kind: Kptfile

# Metadata of the resource described by the kptfile.
metadata:
    # Name of this resource.
    name: resource-name
    # Namespace of this resource.
    namespace: resource-namespace

# The upstream locator for a package.
upstream:
    # The type of origin for a package. Only supports "git."
    type: git

    # The locator for a package on Git.
    git:
        # The repository of the package.
        repo: myrepo
        # The sub directory in the git repository.
        directory: /mydir/mysubdir
        # A git branch, tag, or a commit SHA-1.
        ref: de3c08

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
        - image: imagerepo.io/functions/my-mutator:latest
          # An inline KRM resource used as the function config.
          # Cannot be specified alongside config or configMap.
          configPath: configPath.yaml
        
    # A list of KRM functions that validate resources.
	# Validators are not permitted to mutate resources.
    validators:
          # Specifies the function container image.
          # Can be fully qualified or infer the host path from
          # kpt's configuration (defaults to gcr.io/kpt-fn)
          # The following resolves to gcr.io/kpt-fn/validate-resources:
        - image: validate-resources
          # An inline KRM resource used as the function config.
          # Cannot be specified alongside config or configMap.
          configPath: configPath.yaml

# Optional information about the package not passed to kpt.
info:
    # The URL of the package's homepage.
    site:
        app.example.com

    # The list of emails of the package authors.
    emails:
        - a@example.com
        - b@example.com

    # SPDX license identifier. See: https://spdx.org/licenses/
    license: Apache-2.0"

    # Relative slash-delimited path to the license file.
    licenseFile: LICENSE.txt

    # The path to documentation about the package.
    doc: README.md

    # A short description of the package.
    description: A package that can be applied anywhere.

    # A list of keywords for this package.
    keywords:
        - super cool
        - awesome package

    # The path to documentation about the package
    man: manual.txt

# Parameters for the inventory object applied to a cluster.
# All of the parameters are required if any are set.
inventory:
    # Name of the inventory resource.
    name: foo

    # Namespace for the inventory resource.
    namespace: test-namespace

    # Unique label to identify inventory resource in cluster.
    inventoryID: SSSSSSSSSS-RRRRR

# The resolved locator for the last fetch of the package.
upstreamLock:
    # The type of origin for a package. Only supports "git."
    type: git

    # The locator for a package on Git.
    git:
        # The repository of the package that was fetched.
        repo: myrepo
        # The sub directory in the git repository that was fetched.
        directory: /mydir/mysubdir
        # The git branch, tag, or a commit SHA-1 that was fetched.
        ref: my-branch
        # SHA-1 for the last fetch of the package.
        commit: de3c08