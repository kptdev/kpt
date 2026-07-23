# Multi-Repo Upgrade Utility

Multi-repository upgrade and dependency maintenance tool for the kptdev ecosystem.

Coordinates Go version upgrades, linter version upgrades, and cross-repository dependency updates across 4 interdependent repositories:

```
krm-functions-sdk  (leaf — no upstream deps)
        ↓
    ┌───┴───────────────┐
    ↓                   ↓
   kpt          krm-functions-catalog
    ↓                   ↓
    └───────┬───────────┘
            ↓
          porch          (top — depends on all three)
```

## Quick Start

```bash
# Upgrade Go version across all repos
./upgrade.sh go-version

# Upgrade a single repo
./upgrade.sh go-version --repo=porch

# Full upgrade: Go + lint + cross-deps
./upgrade.sh all

# Full upgrade + push PRs (one-shot)
./upgrade.sh all --push

# Regenerate catalog documentation
./upgrade.sh generate-docs

# Two-step workflow: upgrade, review diffs, then push separately
./upgrade.sh go-version
# ... inspect git diff in workspace/ ...
./upgrade.sh push --for=go-version
```

## Subcommands

| Subcommand | Description |
|---|---|
| `go-version` | Bump `go` directive in all `go.mod` files, then verify (tidy, fmt, vet, build) |
| `lint-version` | Bump `GOLANGCI_LINT_VERSION` in Makefiles |
| `cross-deps` | Upgrade dependencies owned by other repos in the set to their latest version |
| `generate-docs` | Generate/sync Hugo doc pages in krm-functions-catalog |
| `all` | Run `go-version` + `lint-version` + `cross-deps` sequentially |
| `push` | Create branch, commit, push, and raise draft PR for pending workspace changes |

## Options

| Option | Description |
|---|---|
| `--repo=NAME` | Scope to a single repository |
| `--continue` | Don't fail-fast; accumulate errors and report at end |
| `--push` | After successful operations, create branch, commit, push, and raise draft PR |
| `--force`, `-f` | Override upstream push protection (allows push to `kptdev`) |
| `--for=CMD` | With `push` subcommand: specify which upgrade was done (default: `all`) |

## Configuration

Edit `config.env` to change target versions, repositories, or exclusions.

### Fork Owner

The `FORK_OWNER` variable controls which GitHub org/user to clone from. All repo URLs are derived from it. Defaults to `Nordix` (shared development forks). PRs always target upstream `kptdev/*` regardless of fork owner.

If `FORK_OWNER` is set to `kptdev`, the script will refuse to push branches to prevent accidental upstream modifications. The check also inspects the actual `origin` remote URL of the cloned workspace, so it catches stale workspaces previously cloned from upstream even if `FORK_OWNER` has since changed. Use `--force` to override this protection.

```bash
# Use your personal fork
FORK_OWNER=myuser ./upgrade.sh go-version

# Default: uses Nordix forks
./upgrade.sh go-version

# Push to kptdev (blocked by default)
FORK_OWNER=kptdev ./upgrade.sh all --push        # ← blocked
FORK_OWNER=kptdev ./upgrade.sh all --push --force # ← allowed
```

### Target Versions

```bash
TARGET_GO_VERSION="1.26.5"
TARGET_GOLANGCI_LINT_VERSION="2.12.2"
```

### Repository Format

```
NAME|GIT_URL|BRANCH|PR_TARGET
```

- `NAME` — identifier used with `--repo` and in output
- `GIT_URL` — SSH clone URL
- `BRANCH` — branch to clone and base for PRs
- `PR_TARGET` — GitHub `owner/repo` for cross-fork PRs (empty = PR against same repo)

### Exclusions

- `EXCLUDE_MODULES` — glob patterns for modules to skip entirely
- `EXCLUDE_DEPS_GLOBAL` — dependencies never upgraded by `cross-deps`
- `EXCLUDE_DEPS_REPO` — per-repo dependency exclusions

## How It Works

1. **Clone** repos into `workspace/` (reuses existing clones)
2. **Clean state** — ensures base branch, discards leftover changes, detects pollution
3. **Run subcommand** — modifies files, verifies each module
4. **Push** (if `--push`) — creates dated branch, commits, pushes, raises draft PR

### Verification Pipeline

Each module is verified with `GOWORK=off`:
```
go mod tidy → go fmt ./... → go vet ./... → go build ./...
```

Modules are built standalone to respect their own dependency pins, avoiding false conflicts from workspace version unification.

### Cross-Deps Resolution

For each module, dependencies owned by another repo in the set are upgraded to the latest published version. Resolution order:
1. GitHub releases API
2. Go module proxy (stable tags, excluding alpha/dev)
3. Go module proxy (all tags)
4. `@latest` pseudo-version

## File Structure

```
scripts/go-upgrade/
├── upgrade.sh          # Main entry point
├── config.env          # Configuration (versions, repos, exclusions)
├── README.md           # This file
└── lib/
    ├── common.sh       # Logging, colors, failure tracking, helpers
    ├── workspace.sh    # Clone, clean state, module discovery
    ├── verify.sh       # tidy/fmt/vet/build pipeline
    ├── versions.sh     # go-version + lint-version subcommands
    ├── deps.sh         # cross-deps subcommand
    └── push.sh         # Branch/commit/push/PR creation
```

## Design Decisions

- **`GOWORK=off` everywhere** — each module is built standalone respecting its own `go.mod` pins
- **No sequential releases needed for Go bumps** — each repo builds independently; PRs can be merged in parallel
- **Cross-fork PRs** — pushes to fork (`Nordix` or personal), raises PRs against upstream `kptdev/*` via GraphQL `createPullRequest` mutation with `headRepositoryId`
- **Fail-fast by default** — first failure stops execution (override with `--continue`)
- **Draft PRs** — all PRs are created as drafts for review before merge
- **Signed commits** — all commits use `--signoff` for DCO compliance

## Prerequisites

- `go` (matching `TARGET_GO_VERSION`)
- `jq`
- `gh` (GitHub CLI, only needed with `--push`)
- SSH access to configured repositories
