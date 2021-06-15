# FAQ

### What is the roadmap for kpt?

Please visit the [roadmap document] and the [kpt milestones].

### What does kpt provide that git clone doesn't?

kpt enables out-of-the-box workflows that `git clone` does not. Such as: cloning
and versioning git subdirectories, updating from upstream by performing
structured merges on resources, programmatically editing configuration (rather
than with an editor), etc.

### How is kpt different from other solutions?

Rather than expressing configuration as code, kpt represents configuration
packages as data, in particular as YAML or JSON objects adhering to [The
Kubernetes Resource Model].

### What's the difference between kpt and kustomize?

[Kustomize] is a CNCF project that is a part of Kubernetes. While kpt and
kustomize support customization of KRM manifests without use of templates or
DSL(s), there are important differences in both feature set and the scope of
these projects. Kpt supports end-to-end solutions for packaging, customization
and actuation of resources. For kustomize, packaging is explicitly out of scope
and application is deferred to kubectl.

Here are the strengths and investment areas for these products:

_kpt_

- Supports both in place and out-of-place customization, kustomize is only.
  focused on providing out-of-place hydration using the [overlay pattern].
- Allows you to edit the configuration in-place without creating complex patches
- Focuses on rebase with resource merge strategy allowing for edited config to
  be updated.
- Has a continuous learning curve as you usually start small with modifying
  several YAML files using an editor and then want to scale with complexity of
  the application.
- Enables workflows that combine programmatic changes (functions) with manual
  edits

_kustomize_

- Treats base layers as immutable
- Programmatic changes (plugins) do not have to be idempotent
- Provide overlays and components that assemble that “build” the final
  configuration

### Do kpt and kustomize work together?

The goal of kpt project is to provide a seamless UX spanning packaging,
hydration, and live functionality. At the same time, kpt follows a modular
design principle to make it possible to use each of its three command groups
(pkg, fn, live) independently if needed. For example:

- You can use packaging without declaring functions
- You can use imperative functions to operate on vanilla directories of k8s
  resources
- You can use apply logic without buying into full the packaging story (still
  need a minimal `Kptfile` though)

We have created a [kustomize solution] which allows you to use kpt for packaging
and actuation, and kustomize for customization.

### Why resource configuration as the artifact rather than templates or configuration DSLs?

As explained in [Declarative application management in Kubernetes], using
resource configuration provides a number of desirable properties:

1. it clearly **represents the intended state** of the infrastructure -- no for
   loops, http calls, etc to interpret

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

6. it **supports customizing generated resources** so the templates don't need
   to be modified

   - artifacts generated from templates or DSLs may be modified directly, and
     then merged when they are regenerated to keep the modifications.

7. it **supports display in UI and tools** which use either OpenAPI or the
   YAML/JSON directly.

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

[the kubernetes resource model]:
  https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/resource-management.md
[declarative application management in kubernetes]:
  https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md
[functions]: /reference/cli/fn/run/
[using functions]: /book/04-using-functions/
[contact]: /contact/
[functions catalog]: https://catalog.kpt.dev/
[roadmap document]:
  https://github.com/GoogleContainerTools/kpt/blob/main/docs/ROADMAP.md
[kpt milestones]: https://github.com/GoogleContainerTools/kpt/milestones
[kustomize solution]:
  https://github.com/GoogleContainerTools/kpt/tree/main/package-examples/kustomize
[kustomize]: https://kustomize.io
[workflow]: /book/02-concepts/02-workflows
[overlay pattern]:
  https://github.com/kubernetes-sigs/kustomize/tree/master/examples/multibases
