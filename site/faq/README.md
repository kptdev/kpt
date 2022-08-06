# FAQ

### What is the roadmap for kpt?

Please visit the [roadmap document] and the [kpt milestones].

### How is kpt different from other solutions?

Think of configuration as an API or data in a database. kpt can operate on configuration
in storage, git or OCI.

Rather than expressing configuration AS code or templates that generate configuration,
kpt represents [Configuration as Data]. In particular, it represents configuration
as YAML or JSON objects adhering to [The Kubernetes Resource Model], the same as the
live state in Kubernetes, which enables novel remedies to configuration drift.

kpt uses an in-place transformation approach to customization: 
read configuration in, modify it, and write it back.

This enables interoperability of a variety of generators,
transformers, and validators. One doesn’t have to make all changes through a monolithic
generator implementation.

By storing the result of config generators and transformers, automated mutations can be separated in time
from use and other modifications. This enables generation via a UI, for example.
It also is one enabler of in-place edits rather than patches or other programmatic overrides.

Another ingredient in the secret sauce is the ability to upgrade from an upstream package
despite downstream modifications. Conceptually it's like deriving and applying patches automatically.

Combine these capabilities with the ability to operate on packages in bulk via APIs,
and new operational capabilities are enabled. 

### What's the difference between kpt and kustomize?

While both kpt and kustomize support customization of KRM resources via a transformation-based approach,
there are important differences in both feature sets and the scopes of these projects. 

_kustomize_

- Builds the final configuration out of place, primarily using the [overlay pattern].
- Treats base layers as immutable, but enables nearly arbitrary overrides.

_kpt_

- Optimizes for WYSIWYG configuration and in-place customization.
- Allows edits to the configuration in-place without creating complex patches.
- Supports rebase with resource merge strategy allowing for edited config to
  be updated.
- Enables workflows that combine programmatic changes ([functions]) with manual
  edits.
- Aims to support mutating and validating admission control on derived packages.
- Also supports packages, [package orchestration], resource actuation, and GitOps.

### Do kpt and kustomize work together?

The goal of kpt project is to provide a seamless UX spanning packaging,
transformation, and actuation functionality. At the same time, kpt follows a modular
design principle to make it possible to use each of its functionality
independently if needed. For example:

- You can use packaging without declaring functions
- You can use imperative functions to operate on vanilla directories of Kubernetes
  resources
- You can use apply logic without buying into the full packaging story (still
  need a minimal `Kptfile` though)

We have created a [kustomize solution] which allows you to use kpt for packaging
and actuation, and kustomize for customization.

### Why resource configuration as the artifact rather than templates or configuration DSLs?

As explained in [Declarative application management in Kubernetes], using
resource configuration provides a number of desirable properties:

1. it clearly **represents the intended state** of the infrastructure

2. it **aligns with how tools developed by the Kubernetes project are written**
   -- `kubectl`, `kustomize`, etc

3. it enables **composition of different types of tools written in different
   languages**

   - any modern language can manipulate YAML / JSON structures, no need to adopt
     `go`

4. it **supports static analysis and validation**

   - develop tools and processes to perform validation and linting

5. it **supports programmatic modification**

   - develop CLIs and UIs for working with configuration rather than using `vim`

6. it **supports display in UI and tools** which use either OpenAPI or the
   YAML/JSON directly.

For a more complete explanation, see the [rationale].

### I really like DSL / templating solution X. Can I use it with kpt?

Yes. kpt supports plugging in solutions which generate or manipulate
configuration, e.g. from DSLs and templates. This may be performed using the
[Functions Catalog]. The generated output may be modified directly, and merged
when regenerated.

### I want to write high-level abstractions like CRDs, but on the client-side. Can I do this with kpt?

Yes. kpt's architecture facilitates the developing programs which may generate
or modify configuration. See the [Using Functions] for how to compose multiple
programs together.

### What's with the name?

kpt was inspired by `apt`, but with a Kubernetes focus. We wanted to uphold the
tradition of naming tools to start with `k`, and also be short enough that you
don't have to alias it. It is pronounced "kept".

### I still have questions. How do I contact you?

[Please reach out!][contact]

[Configuration as Data]:
  https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/06-config-as-data.md
[package orchestration]:
  https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/07-package-orchestration.md
[the kubernetes resource model]:
  https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md
[declarative application management in kubernetes]:
  https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/declarative-application-management.md
[rationale]: https://kpt.dev/guides/rationale
[functions]: /reference/cli/fn/eval/
[using functions]: /book/04-using-functions/
[contact]: /contact/
[functions catalog]: https://catalog.kpt.dev/
[roadmap document]:
  https://github.com/GoogleContainerTools/kpt/blob/main/docs/ROADMAP.md
[kpt milestones]: https://github.com/GoogleContainerTools/kpt/milestones
[kustomize solution]:
  https://github.com/GoogleContainerTools/kpt/tree/main/package-examples/kustomize
[kustomize]: https://kustomize.io
[overlay pattern]:
  https://github.com/kubernetes-sigs/kustomize/tree/master/examples/multibases
