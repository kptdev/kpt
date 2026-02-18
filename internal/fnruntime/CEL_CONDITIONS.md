# CEL Conditional Function Execution

Functions can be executed conditionally using CEL expressions in the Kptfile.

## Usage

```yaml
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-namespace:v0.4
    condition: "resources.exists(r, r.kind == 'ConfigMap')"
    configMap:
      namespace: production
```

The function runs only if the condition evaluates to true.

## Examples

Check if a resource exists:
```
resources.exists(r, r.kind == "Deployment")
```

Check resource count:
```
resources.filter(r, r.kind == "Service").size() > 0
```

Check nested fields:
```
resources.exists(r, r.kind == "Deployment" && r.spec.replicas > 3)
```

## Implementation Notes

- CEL environment and expression are compiled once per function
- Expressions are evaluated on each function invocation
- Maximum expression length: 10,000 characters
- Expressions must return boolean
