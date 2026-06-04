# Cloud Agent Instructions for kpt

For Copilot, Cursor, Codex, Gemini Code Assist or any agents generating code or review Pull Requests
in this repository. 

## Generic rules

### Coding

* Never commit directly to this repository, always aim for a pull request
* Always ask for a final review from a human before committing
* Make sure that AI agent usage is attributed in the correct way

### Reviewing PRs

* When comitting suggested changes add a signoff of the approving human and add the `Assisted-by`
  tag to the commit message
* If a comment was not accepted and the Conversation was resolved do not make the same comment again


## Repository Overview

**kpt** is a package-centric toolchain that automates Kubernetes configuration editing and management. It enables declarative configuration authoring, automation, and delivery at scale through a "Configuration as Data" approach, supporting Kubernetes platforms and KRM-driven infrastructure (e.g., Config Connector, Crossplane).

- **Language**: Go 1.26.3
- **Repository Size**: ~78 MB
- **License**: Apache 2.0
- **Key Topics**: Kubernetes, configuration management, CLI tooling, GitOps, policy-as-code, KRM (Kubernetes Resource Model)

## Quick Build & Test Commands

**Trust these instructions first.** Only perform searches if you find this information incomplete or inaccurate.

### Prerequisites
- Go 1.26.3 (specified in `go.mod`)
- Git (required and checked at runtime)
- Docker or Podman (for `test-docker` and function runtime tests)
- KinD (CI uses v0.30.0; `make install-kind` installs v0.29.0) (for e2e live apply tests with Kubernetes 1.33 and 1.34)

### Build

```bash
make build
```
This compiles kpt to `$(go env GOPATH)/bin/kpt` using LDFLAGS with git commit SHA.

### Run All Checks (Build, Test, Lint)

```bash
make all
```

Runs: `fix vet fmt lint test build tidy` in sequence. *Always* run this before committing.

### Unit Tests

```bash
make test
```

Runs Go tests with coverage. Set `KRM_FN_RUNTIME` to select function runtime (docker/podman, default uses system default).

### Docker/Podman Runtime Tests

```bash
make test-docker
```

Requires Docker or Podman. Tests that need container runtime (e.g., pipeline tests). Respects `KRM_FN_RUNTIME` environment variable.

### E2E Function Tests

#### Render tests

```bash
make test-fn-render T=".*"
```
#### Eval tests

```bash
make test-fn-eval T=".*"
```

Use `T` parameter to filter tests by regex (e.g., `T=fnconfig` for function config tests). Set `KRM_FN_RUNTIME` to select runtime.

### E2E Live Apply Tests

```bash
make test-live-apply
```

Requires KinD with Kubernetes 1.33.4 and 1.34.0 (specific SHAs pinned in CI). These tests use Kind internally. Timeout is 20 minutes.

### Linting

```bash
make lint
```

Runs golangci-lint v2.11.4. If already installed locally with matching version, uses it; otherwise downloads and runs via `go run`.

### Code Generation & Formatting

```bash
make fmt        # Run gofmt and goimports
make fix        # Run go fix
make vet        # Run go vet
make tidy       # Run go mod tidy
make generate   # Generate code from templates (mdtogo, copyright headers)
make schema     # Generate schema
```

### Setting Up Git Config (Required for Tests)

Many tests require git configuration:

```bash
git config --global user.email "you@example.com"
git config --global user.name "Your Name"
```

## Project Layout

### Root Level Files & Directories

* **main.go**: CLI entry point; contains //go:generate directives for CLI documentation
* **Makefile**: Primary build orchestration
* **.golangci.yml**: Linting configuration (golangci-lint v2.11.4)
* **go.mod / go.sum**: Dependency management (Go 1.26.3)
* **CONTRIBUTING.md**: Contribution guidelines and code review requirements
* **CODEOWNERS**: Default reviewers

### Key Source Directories

* **commands/**: CLI command implementations using Cobra framework
* **run/**: Main CLI setup; contains GetMain() that initializes Cobra commands with environment setup
* **pkg/**: Core library packages (business logic, utilities)
* **internal/**: Internal packages; includes internal/docs/generated/ (generated from Markdown via mdtogo)
* **mdtogo/**: Code generator tool that converts CLI documentation Markdown files to Go variables

### Build & CI Configuration

* **.github/workflows/go.yml**: Main CI workflow
  * Runs on Linux (docker/podman matrix) and macOS
  * Executes: `make all` + `make test-docker`
  * Triggered on PRs and pushes (excludes dependabot branches)
* **.github/workflows/verifyContent.yml**: Verifies CLI examples
  * Runs `make build`, installs mdrip/kind, runs `make site-verify-examples`
  * Triggered on changes to `commands/`, `internal/` paths
* **.github/workflows/e2eEnvironment.yml**: KinD-based e2e tests
  * Tests Kubernetes 1.33 and 1.34 with KinD v0.30.0
  * Runs `./e2e/live/end-to-end-test.sh -k <K8S_VERSION>`
* **.github/workflows/live-e2e.yml**: Live apply e2e tests
  * Tests with pinned Kubernetes image SHAs
  * Runs `make test-live-apply` with `K8S_VERSION` environment variable

### Documentation & Tests

* **documentation/**: Hugo-based website published to kpt.dev
  * Run `make serve` from root (serves docs locally)
  * Requires `npm install` first

* **e2e/**: End-to-end test suites
  * Contains testdata directories for function render/eval tests
  * Live tests in `e2e/live/end-to-end-test.sh`

### Other Notable Directories

* **release/**: Release automation (GoReleaser config, Homebrew formula generation)
* **hack/**: Miscellaneous development utilities
* **healthcheck/**: Separate module for health checking (Go 1.26.3, local Makefile)
* **thirdparty/**: Third-party code (excluded from linting)
* **Formula/**: Homebrew package definition (generated by `go run ./release/formula/main.go VERSION`)

## Linting Rules & Style

### Linter Configuration

* **Enabled Linters**: bodyclose, copyloopvar, dogsled, dupl, errcheck, gochecknoinits, goconst, gocritic, gocyclo, govet, ineffassign, lll, misspell, nakedret, revive, staticcheck, unconvert, unparam, unused, whitespace
* **Duplication Threshold**: 400 lines
* **Cyclomatic Complexity**: max 30
* **Line Length**: max 170 characters
* **Revive Confidence**: 0.85
* **Excluded Paths**: thirdparty/, third_party, builtin, examples
* **Test Files**: Further relaxed rules (gosec, funlen disabled for *_test.go files)

### Code Style Requirements (from CONTRIBUTING.md)

* **Copyright Headers**: All files must have Apache 2.0 license header
  * Use format: // Copyright YEAR The kpt Authors (or year range if modified)
   * Year should match creation year, or creation-to-modification year range
* **Developer Certificate of Origin (DCO)**: Commits must be signed with -s flag
* **Large Features**: Require reviewed and merged design document (use /docs/design-docs/00-template.md as template)
* **AI Usage**: Must declare AI usage in PR description and commit messages with `Assisted-by: AGENT_NAME:MODEL_VERSION` format

## Validation Checklist for PRs

Before submitting a PR, verify:

1. ✅ All tests pass: make all
1. ✅ All linting passes: make lint
1. ✅ Code formatted: make fmt
1. ✅ Dependencies tidied: make tidy
1. ✅ Copyright headers added/updated per CONTRIBUTING.md
1. ✅ DCO sign-off: use git commit -s
1. ✅ For CLI/API changes: design document reviewed and merged
1. ✅ AI usage declared in PR description (if applicable)

## Common Build Issues & Workarounds

### Docker/Podman Not Available

If `make test-docker` fails due to missing Docker/Podman:

* Install Docker Desktop or Podman
* For Podman: ensure it's on PATH and `podman version` runs successfully
* Set `KRM_FN_RUNTIME=podman` if using Podman

### KinD Setup Issues

For e2e live tests (`make test-live-apply`):

* KinD v0.30.0 is auto-installed by CI workflow
* Requires Docker running in background
* Tests use specific Kubernetes image SHAs (see live-e2e.yml matrix)
* Timeout is 20 minutes; allow sufficient time

### Git Configuration

Tests fail silently if git user not configured. Always run:

```bash
git config --global user.email "you@example.com"
git config --global user.name "Your Name"
```

### Module Generation

If changes modify CLI documentation in documentation/content/en/reference/cli/:

* Run `make generate` to regenerate `internal/docs/generated/`
* Commit generated files

### Known CI Skips

* Windows build currently disabled (see `.github/workflows/go.yml` line 88-104, issue #3463)
* Some linters disabled: `funlen`, `gosec` (marked TODO in `.golangci.yml`)

## Environment Variables

* **KRM_FN_RUNTIME**: Select function runtime for tests: `docker`,  `podman` or `nerdctl`
* **K8S_VERSION**: Kubernetes version for e2e live tests (used in CI with pinned SHAs)
* **KPT_NO_PAGER_HELP**: Set to 1 to disable pager for help output
PAGER: Custom pager command (default: less -R)
* **KPT_FN_WASM_RUNTIME**: WASM function runtime selection
* **GOPATH**: Go workspace path (used in CI workflows)
* **GOBIN**: Go binary installation directory

## Key Dependencies

* **Cobra**: CLI framework
* **Kubernetes libraries (k8s.io/*)**: For Kubernetes resource handling
* **sigs.k8s.io/cli-utils**: CLI utilities
* **sigs.k8s.io/kustomize/kyaml**: YAML handling for Kubernetes
* **sigs.k8s.io/controller-runtime**: Controller and reconciliation patterns
* **wasmtime-go**: WebAssembly function runtime
* **go-containerregistry**: Container image handling

## Testing Strategy

* **Unit Tests**: Run with `make test` (standard Go tests)
* **Docker-based Tests**: Run with `make test-docker` (requires container runtime)
* **Function E2E Tests**: Run with make `test-fn-render` / `make test-fn-eval` (testdata-driven)
* **Live Apply E2E**: Run with `make test-live-apply` (KinD-based, 20-minute timeout)
* **Example Verification**: Run with `make site-verify-examples` (verifies CLI examples in docs)

## Implementation Notes for Code Changes

* **CLI Commands**: Add to `commands/` directory; use Cobra; update `documentation/cli` for documentation
* **Library Code**: Place in `pkg/` for public APIs; use `internal/` for internal utilities
* **Tests**: Colocate `*_test.go` files with source; use testdata directories for fixtures
* **Generated Code**: Run `make generate` after modifying templates
* **Linting Issues**: Address all `golangci-lint` findings; consult `.golangci.yml` for thresholds
* **Git Flow**: Always create feature branches; use squash-merge preferred per repository settings
* **Documentation**: Update Markdown in documentation/; run `make serve` to make sure that there are no errors

Trust these instructions. They have been validated against the Makefile, GitHub workflows, go.mod,
and contributing guidelines. Only search for additional details if:

* Build or test commands fail with unexpected errors
* Instructions reference non-existent paths or commands
* New tool versions are released and compatibility is unclear
