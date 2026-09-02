---
title: Testing
linkTitle: Testing
description: Golden tests and unit tests for KRM functions.
toc_hide: false
menu:
  main:
    parent: "KRM Function Developer Guide"
    weight: 30
---
The SDK provides a golden test framework in `fn/testhelpers` for snapshot-based
testing of KRM functions.

## Golden Test Pattern

Golden tests compare the function output against the expected baseline files. This
approach catches regressions and makes it easy to review output changes.

### Directory Structure

```
testdata/
├── test-case-1/
│   ├── _expected.yaml    # Expected output (full ResourceList YAML)
│   ├── _fnconfig.yaml    # FunctionConfig for this test case
│   └── resources.yaml    # Input KRM resources
└── test-case-2/
    ├── _expected.yaml
    ├── _fnconfig.yaml
    └── resources.yaml
```

Conventions:
- Files prefixed with `_` are special — they are not included in the input items.
- `_fnconfig.yaml` contains the functionConfig passed to your function.
- `_expected.yaml` contains the expected ResourceList output.
- All other `.yaml`/`.yml` files (and a `Kptfile`, if present) are parsed as input
  resources. Files with any other extension are ignored.
- You can have multiple input files (e.g., `deployments.yaml`, `services.yaml`).
  Input files are read in sorted (alphabetical) order, so the assembled item
  ordering is deterministic.

### Writing a Golden Test

```go
package main

import (
    "context"
    "testing"

    "github.com/kptdev/krm-functions-sdk/go/fn"
    "github.com/kptdev/krm-functions-sdk/go/fn/testhelpers"
)

func TestFunction(t *testing.T) {
    runner := fn.WithContext(context.TODO(), &SetLabels{})
    testhelpers.RunGoldenTests(t, "testdata", runner)
}
```

`RunGoldenTests` will:
1. Discover all subdirectories under `testdata/`.
2. For each subdirectory, parse all non-`_` prefixed YAML files as input items.
3. Parse `_fnconfig.yaml` as the functionConfig.
4. Run your processor against the assembled ResourceList.
5. Compare the output against `_expected.yaml`.

### Example Test Data

The following example is illustrative — it shows what test data looks like for a
function that sets labels. The [`go/get-started/`](../go/get-started/) example
provides a minimal working skeleton you can build from.

`testdata/add-labels/_fnconfig.yaml`:
```yaml
apiVersion: fn.kpt.dev/v1alpha1
kind: SetLabels
metadata:
  name: my-config
labels:
  app: my-app
```

`testdata/add-labels/resources.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app
```

`testdata/add-labels/_expected.yaml`:
```yaml
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: v1
  kind: Service
  metadata:
    name: my-service
    labels:
      app: my-app
  spec:
    selector:
      app: my-app
functionConfig:
  apiVersion: fn.kpt.dev/v1alpha1
  kind: SetLabels
  metadata:
    name: my-config
  labels:
    app: my-app
results:
- message: updated labels
  severity: info
```

## Running Tests

From your function's module root (where `go.mod` lives):

```bash
go test ./...
```

If the function output does not match `_expected.yaml`, the test fails with a
diff showing what changed. See [`go/get-started/`](../go/get-started/) for a
complete working example.

## Updating Expected Output

When your function's output changes intentionally, regenerate the expected files:

```bash
WRITE_GOLDEN_OUTPUT=1 go test ./...
```

This overwrites any `_expected.yaml` that differs from the actual output. Any
non-empty value enables write mode (`WRITE_GOLDEN_OUTPUT=1`, `=true`, etc.).
Note that the run which writes a golden file is reported as a **test failure**
(`wrote output to ...`) — this is intentional, so a rewrite never silently
passes in CI. Re-run the tests without the env var to confirm they pass, and
review the diffs in version control before committing.

**Caution:** `WRITE_GOLDEN_OUTPUT` accepts whatever the function currently
produces as "correct." If the function has a bug, you have just blessed buggy
output. Golden tests verify *stability* (did the output change?), not
*correctness* (is the output right?). Always review the diffs carefully.
For correctness guarantees, complement golden tests with property-based tests
that assert invariants (e.g., "all resources have the expected label").

Note: other kpt ecosystem projects use different env var names for the same
purpose (`KPT_E2E_UPDATE_EXPECTED` in kpt, `UPDATE_GOLDEN_FILES` in porch).
`WRITE_GOLDEN_OUTPUT` is the standard for the SDK and catalog functions.

## Testing a ResourceListProcessor

`RunGoldenTests` accepts any `fn.ResourceListProcessor`, so it works with both
`fn.Runner` (wrapped via `fn.WithContext`) and direct `ResourceListProcessor`
implementations:

```go
func TestGenerator(t *testing.T) {
    testhelpers.RunGoldenTests(t, "testdata", &MyGenerator{})
}
```

## Unit Testing Without Golden Files

For simpler unit tests, you can construct a ResourceList directly:

```go
func TestSetLabels(t *testing.T) {
    input := []byte(`
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: test
functionConfig:
  apiVersion: fn.kpt.dev/v1alpha1
  kind: SetLabels
  labels:
    env: prod
`)
    runner := fn.WithContext(context.TODO(), &SetLabels{})
    output, err := fn.Run(runner, input)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    rl, err := fn.ParseResourceList(output)
    if err != nil {
        t.Fatalf("failed to parse output: %v", err)
    }

    label, _, _ := rl.Items[0].NestedString("metadata", "labels", "env")
    if label != "prod" {
        t.Errorf("expected label env=prod, got %q", label)
    }
}
```

## Tips

- Keep test cases focused — one behavior per test directory.
- Use descriptive directory names (e.g., `empty-input`, `missing-namespace`, `multiple-resources`).
- The `_fnconfig.yaml` can be empty if your function doesn't require configuration.
- Golden tests also catch unintentional formatting changes. This helps to maintain a stable output.

## End-to-End Testing

The SDK's `testhelpers.RunGoldenTests` tests function logic in isolation — no
container, no kpt CLI. For full integration testing (container execution,
`kpt fn eval`/`kpt fn render` pipelines), the kpt repo provides a separate e2e
test runner at
[`pkg/test/runner`](https://github.com/kptdev/kpt/tree/main/pkg/test/runner).

The e2e runner uses a different test structure (`.expected/` directories with
`config.yaml`, `diff.patch`, `results.yaml`) and is used by the
[krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog) `tests/`
directory to validate the functions running inside the containers against `kpt fn render`.

---

Next: [Containerizing]({{% relref "/guides/krm-functions/containerizing" %}}) — packaging your function as a container image.
