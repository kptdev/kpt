---
title: "Chapter 9: CI user guide"
linkTitle: "Chapter 9: CI user guide"
description:
  This chapter provides a practical guide for using kpt in CI/CD workflows,
  including rendering (with validators) and gated apply steps.
toc: true
menu:
  main:
    parent: "Book"
    weight: 90
---

## Overview: Using kpt in CI/CD

Continuous integration (CI) is the practice of running automated checks on every change so that teams can validate
configuration early and consistently. In a CI/CD pipeline, kpt fits naturally because it operates on local files and
produces deterministic output. This makes it easy to run in ephemeral build environments and validate changes before
they reach a cluster.

In CI, kpt is typically used to render packages (including validators), and in some workflows to apply configuration in a
gated step. CI does not author or mutate packages; it consumes versioned packages from source control and verifies that they
render correctly and meet policy requirements.

## CI Responsibilities vs. Developer Responsibilities

kpt follows the configuration-as-data model: Git is the source of truth, and packages represent declared intent. The
division of responsibilities exists to preserve that intent and to keep automation predictable. Developers author and
declare intent in the repository, while CI renders (including validators) and optionally applies that intent in controlled
environments.

At developer time (outside CI), teams create and evolve packages. This includes writing the `Kptfile`, declaring
pipelines, and version-controlling configuration. Changes are reviewed and merged in Git so the repository remains the
authoritative record.

At CI time, the pipeline consumes the versioned package, runs the declared pipeline with `kpt fn render`, fails fast when
validators fail, and optionally applies the rendered configuration using `kpt live apply` when explicit gates are satisfied.
CI should never mutate the package source as part of the build; doing so breaks the source-of-truth model and makes
changes harder to audit and reproduce.

| Area               | Developer responsibilities                             | CI responsibilities                                         |
| ------------------ | ------------------------------------------------------ | ----------------------------------------------------------- |
| Source of truth    | Author configuration in Git and manage version history | Treat Git as authoritative input                            |
| Package definition | Create packages and write the `Kptfile`                | Consume packages as-is                                      |
| Pipelines          | Declare `pipeline` steps in the `Kptfile`              | Execute declared pipelines with `kpt fn render`             |
| Validation         | Choose validators and policies                         | Fail fast on validation results                             |
| Apply              | Decide when packages are ready to deploy               | Optionally apply with `kpt live apply` under explicit gates |
| Mutations          | Make intentional edits in Git                          | Do not mutate package sources in CI (anti-pattern)          |

## Typical kpt CI workflow (conceptual)

This section provides a system-agnostic mental model for how kpt is typically used in CI. The flow is intentionally
simple and does not assume any specific CI platform or YAML configuration. The same steps should behave the same way
when run locally or in CI, which helps keep automation deterministic and easy to debug.

At a high level, a CI run follows this sequence:

1. Check out the repository that contains the kpt package.
2. Install kpt in the build environment.
3. Run `kpt fn render` to execute the declared pipeline.
4. Observe results and fail fast if any checks fail.
5. Optionally apply the rendered resources when explicit deployment gates are satisfied.

![img](/images/ci-kpt-workflow.svg)

This flow emphasizes determinism and no hidden state: the repository is the source of truth, the rendered output is
derived entirely from the checked-out files, and the results should be consistent across developer machines and CI
runners.

## Rendering in CI

Rendering is the most important CI step in a kpt workflow. The `kpt fn render` command executes the package pipeline
declared in the `Kptfile`, running mutators and validators in a predictable order. The output is the fully hydrated
configuration that CI can use for downstream steps.

### Prerequisites

Since kpt functions run as containers, your CI environment must have access to a container runtime (for example,
Podman).

Podman is preferred in CI because it supports rootless operation and does not require a daemon.

- Podman socket: Ensure your CI step can access the Podman socket (for example, `/run/podman/podman.sock` or rootless
  `$XDG_RUNTIME_DIR/podman/podman.sock`).
- Privileges: The CI runner requires permissions to pull images and run containers.

### Why render (including validation)

Render is the default CI action because it catches configuration errors early. Validators in the pipeline fail the build
when schemas, policies, or constraints are not met. This makes CI a reliable safety net before any configuration is
applied to a cluster.

Run `kpt fn render` on every change so CI always tests the exact package state stored in Git. This keeps results
deterministic and avoids hidden state between runs.

### `kpt fn render` vs `kpt fn eval`

It is important to distinguish `kpt fn render` from `kpt fn eval`:

- `kpt fn render`: Runs the declared pipeline from the `Kptfile`. Use this in CI to ensure the rendered output matches
  the intent declared in Git.
- `kpt fn eval`: Runs an ad-hoc function. In CI, this is useful for extra validation (like a separate linter step), but
  it does not represent the package's definition.

## Applying configuration in CI (optional and gated)

Applying configuration from CI should be treated as optional and tightly controlled to prevent accidental cluster
changes. In most pipelines, [kpt live apply](/reference/cli/live/apply/) runs only when a deployment is explicitly
authorized.

Run apply only under these conditions:

- The pipeline is executing on the main branch.
- The pipeline is part of a release workflow.
- A manual approval gate has been satisfied.

Avoid running apply on pull requests. PRs are for review and validation, not for changing live clusters. Applying from
unmerged changes makes it difficult to audit what was deployed and can introduce drift between Git and the cluster.

Scope cluster credentials to the minimum permissions needed for the target environment. Use separate credentials for
different environments and avoid sharing production access with non-deployment jobs.

Always render before apply. The `kpt fn render` step produces the exact, validated output that should be deployed, and
it ensures the apply step reflects the intent stored in Git.

Pruning & inventory (critical safety note): `kpt live apply` tracks an inventory of deployed resources and will
prune resources that exist in the cluster but are missing from the current package. Unlike `kubectl apply`, it will
delete resources that are absent from the rendered input. If a CI pipeline accidentally renders an empty directory and
runs apply, it can delete the application. Guard apply with explicit gates and verify rendered output before
deployment.

## Handling secrets in CI

Handling secrets is often the most challenging part of automation. The core principle for kpt is simple: secrets are
runtime inputs, not configuration data. Git is the source of truth for declared configuration, but secrets should never
be stored in the repository.

### The golden rules

- Never in Git: Secrets must not appear in YAML files, the `Kptfile`, or `functionConfig`.
- Runtime only: Inject secrets only at render or apply time, and keep them in memory or a temporary filesystem.

### Where to store secrets

Do not rely on the kpt package to store sensitive data. Use one of the following:

- CI native stores (for example, GitHub Secrets or GitLab CI/CD Variables).
- External vaults (for example, HashiCorp Vault, Google Secret Manager, AWS Secrets Manager, or Azure Key Vault).

### How to inject secrets

Because kpt separates configuration from execution, secrets must be supplied to kpt commands at runtime.

#### Environment variables (for functions)

If a function requires credentials (for example, a validator that calls an external API), pass them as environment
variables. Fetch the secret in a setup step and export it so the container runtime can pass it to the function.

```shell
$ export API_TOKEN=$(vault read -field=token secret/my-api)
```

```shell
$ kpt fn render
```

#### File mounts (for `kpt live apply`)

Applying to a cluster requires credentials such as a kubeconfig file or service account token. Mount the credential
file from your secret store into the CI runner's filesystem and point `kpt live apply` to it.

```shell
$ echo "$KUBECONFIG_CONTENT" > /tmp/kubeconfig
```

```shell
$ KUBECONFIG=/tmp/kubeconfig kpt live apply
```

### Integration with external vaults

When using an external vault, the standard pattern is fetch-then-run:

1. Authenticate: The CI job authenticates to the vault (OIDC, AppRole, or similar).
2. Fetch: The job retrieves only the secrets required for this pipeline.
3. Inject: Secrets are exported as environment variables or written to a tmpfs volume.
4. Execute: kpt runs using the injected credentials.
5. Cleanup: The CI runner is destroyed, wiping the secrets.

## Example: Using kpt in a Cloud Build pipeline

Cloud Build is a concise way to demonstrate the CI pattern, but the same structure applies to any CI system. The
examples below are intentionally small and focus on the critical steps: install, render (including validators), and an
optional gated apply.

To keep the example concrete, we use the WordPress package. You can fetch it locally with:

```shell
$ kpt pkg get https://github.com/kptdev/kpt/package-examples/wordpress@v1.0.0-beta.61
```

### Render-only build

This build renders configuration on every change and runs validators as part of the pipeline. It does not deploy.

```yaml
steps:
  # Install step: install kpt into the Docker builder
  - name: gcr.io/cloud-builders/docker
    entrypoint: bash
    args:
      - -c
      - |
        curl -L https://github.com/kptdev/kpt/releases/download/${_KPT_VERSION}/kpt_linux_amd64 -o /usr/local/bin/kpt
        chmod +x /usr/local/bin/kpt

  # Render step: execute declared pipeline
  # Validators run as part of the pipeline and fail the build when checks fail.
  - name: gcr.io/cloud-builders/docker
    entrypoint: bash
    args:
      - -c
      - |
        kpt fn render ${_PACKAGE_DIR}

substitutions:
  _KPT_VERSION: v1.0.0-beta.61
  _PACKAGE_DIR: wordpress
```

### Deployment build (gated)

This build is intended for main or release workflows and assumes a manual approval gate. It renders first, then
applies only after credentials are injected. The example uses the `package-examples/wordpress` package.

```yaml
steps:
  # Install step
  - name: gcr.io/cloud-builders/docker
    entrypoint: bash
    args:
      - -c
      - |
        curl -L https://github.com/kptdev/kpt/releases/download/${_KPT_VERSION}/kpt_linux_amd64 -o /usr/local/bin/kpt
        chmod +x /usr/local/bin/kpt

  # Render step
  - name: gcr.io/cloud-builders/docker
    entrypoint: bash
    args:
      - -c
      - |
        kpt fn render ${_PACKAGE_DIR}

  # Apply step: gated deployment with secrets
  - name: gcr.io/cloud-builders/docker
    entrypoint: bash
    secretEnv: ["KUBECONFIG_CONTENT"]
    args:
      - -c
      - |
        # Write secret to a file for use
        echo "$$KUBECONFIG_CONTENT" > /workspace/kubeconfig

        # Apply with the kubeconfig
        KUBECONFIG=/workspace/kubeconfig kpt live apply ${_PACKAGE_DIR}

substitutions:
  _KPT_VERSION: v1.0.0-beta.61
  _PACKAGE_DIR: wordpress

# Define where the secret comes from
availableSecrets:
  secretManager:
    - versionName: projects/$PROJECT_ID/secrets/my-kubeconfig/versions/latest
      env: "KUBECONFIG_CONTENT"
```

## Common mistakes and anti-patterns

This section highlights practices that commonly lead to CI failures, drift, or unintended cluster changes. Avoid the
following:

- Running `kpt pkg init` in CI. Packages and `Kptfile` metadata should be authored by developers, not created during
  CI runs.
- Mutating packages in CI. CI should validate and render the declared intent, not change the source of truth.
- Storing secrets in configuration. Secrets must not appear in YAML files, the `Kptfile`, or `functionConfig`.
- Applying on pull requests. PRs should validate only; deployment belongs in gated, mainline workflows.
- Committing rendered output back to the source package. Rendered results are derived artifacts and should not be
  committed to the source repository in CI.
