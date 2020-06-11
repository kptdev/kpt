---
title: "FAQ"
linkTitle: "FAQ"
type: docs
weight: 20
menu:
  main:
    weight: 5
description: >
  Frequently asked questions
---

#### **Q: What's with the name?**

A: `kpt` was inspired by `apt`, but with a Kubernetes focus. We wanted to uphold the tradition
of naming tools to start with `k`, and also be short enough that you don't have to alias it.
It is pronounced "kept".

#### **Q: What does kpt provide that git clone doesn't**

A: `kpt` enables out-of-the-box workflows that `git clone` does not such as:
cloning and versioning git subdirectories, updating from upstream by
performing structured merges on resources, programmatically editing
configuration (rather than with an editor), etc

#### **Q: How is `kpt` different from other solutions?**

A: Rather than expressing configuration as code, `kpt` represents configuration packages as data,
in particular as YAML or JSON objects adhering to the
[Kubernetes resource model]

#### **Q: Why resource configuration as the artifact rather than templates or configuration DSLs?**

A: As explained in [Declarative application management in Kubernetes],
using resource configuration provides a number of desirable properties:

1. it clearly **represents the intended state** of the infrastructure -- no for loops, http calls,
   etc to interpret

2. it **aligns with how tools developed by the Kubernetes project are written** --
   `kubectl`, `kustomize`, etc

3. it enables **composition of different types of tools written in different languages**

   - any modern language can manipulate YAML / JSON structures, no need to adopt `go`

4. it **supports static analysis and validation**

   - develop tools and processes to perform validation and linting

5. it **supports programmatic modification**

   - develop CLIs and UIs for working with configuration rather than using `vim`

6. it **supports customizing generated resources** so the templates don't need to be modified

   - artifacts generated from templates or DSLs may be modified directly, and then merged
     when they are regenerated to keep the modifications.

7. it **supports display in UI and tools** which use either OpenAPI or the YAML/JSON directly.

#### **Q: Isn't writing YAML hard?**

A: `kpt` offers a collection of utilities which enable working with configuration
programmatically to simplify the experience. Using `vi` to edit YAML should be
necessary only for bootstrapping, and the common cases should use [setters]
or [functions] to generate or modify YAML configuration.

#### **Q: I really like DSL / templating solution X. Can I use it with `kpt`?**

A: Yes. `kpt` supports plugging in solutions which generate or manipulate configuration, e.g. from
DSLs and templates. This may be performed using [functions]. The generated
output may be modified directly, and merged when regenerated.

#### **Q: I want to write high-level abstractions like CRDs, but on the client-side. Can I do this with `kpt`?**

A: Yes. `kpt`'s architecture facilitates the developing programs which may generate or modify
configuration. Multiple programs may be composed together. See [functions].

#### **Q: How do I roll out changes throughout my organization using `kpt`?**

A: This can be done one of several ways, including: 1) using semantic versioning or release
channels with [functions], or 2) [updating] packages.

#### **Q: Is there a container image that contains kpt?**

A: Yes. [gcr.io/kpt-dev/kpt] contains the `kpt` binary.

#### **Q: I still have questions. How do I contact you?**

A: [Please reach out!][contact]

###

[updating]: /kpt/reference/pkg/update
[functions]: /kpt/reference/fn/run
[setters]: /kpt/reference/cfg/set
[gcr.io/kpt-dev/kpt]: https://gcr.io/kpt-dev/kpt
[pkg]: /kpt/reference/pkg
[cfg]: /kpt/reference/cfg
[fn]: /kpt/reference/fn
[live]: /kpt/reference/live
[contact]: /kpt/contact
[kubernetes resource model]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/resource-management.md
[declarative application management in kubernetes]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md
