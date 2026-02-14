Hey! I'd like to take this one if it's still available.

I've been looking through the issue and the codebase, and this seems like a really useful feature. The workaround of writing a custom function just to check a condition is definitely awkward when you just want to do something simple like set-annotation.

Here's what I'm thinking for the implementation:

**Approach:**
- Add a `condition` field to the Function struct in the pipeline schema
- Use the CEL library (google/cel-go) for evaluating expressions
- The runner will check the condition before executing each function
- If condition evaluates to false, skip the function (just pass through the input unchanged)

**Example of what it would look like:**
```yaml
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-namespace:v0.4
    condition: "resources.exists(r, r.kind == 'ConfigMap' && r.metadata.name == 'env-config')"
    configMap:
      namespace: production
```

I'll start with the schema changes and get CEL integrated into the runner. Should have something working soon.

Let me know if there are any specific considerations I should keep in mind!
