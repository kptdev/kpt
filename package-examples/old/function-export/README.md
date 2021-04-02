# Function Export Example

The package is part of the KPT Function Export Guide:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions from the [KPT Functions Catalog](../../catalog) declaratively:
  - `label-namespace` adds a label to all Namespaces.
  - `gatekeeper-validate` enforces constraints over all resources.
