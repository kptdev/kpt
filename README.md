# KPT

Kubernetes Platform Toolchain

- Publish, Consume and Update packages of Kubernetes Resource Configuration.
- Develop and Update Configuration programmatically.
- Filter and Display Configuration packages.
- Apply Configuration to clusters.

`kpt` combines package management commands with upstream Kubernetes tools to provide a complete
toolchain for building platforms for Kubernetes Resources.

## Installation

Binaries:

- [darwin](https://storage.cloud.google.com/kpt-dev/kpt.master_darwin_amd64)
- [linux](https://storage.cloud.google.com/kpt-dev/kpt.master_linux_amd64)
- [windows](https://storage.cloud.google.com/kpt-dev/kpt.master_windows_amd64)

Source:

    GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt

or

    git clone https://github.com/GoogleContainerTools/kpt && cd kpt && make

### [Documentation](docs/README.md)

See the [docs](docs/README.md) for more information on how to use `kpt`.

## FAQ

### **Q: How is `kpt` different from other solutions?**

A: Rather than developing configuration by expressing it as code, `kpt` develops packages
   using the API native format -- i.e. as json or yaml objects adhering to the API schema.

### **Q: Why Resource configuration as the artifact rather than Templates or DSLs?**  

A: Using Resource configuration provides a number of desirable properties:

  1. it clearly **represents the intended state** of the infrastructure -- no for loops, http calls,
    etc to interpret

  2. it **aligns with how tools developed by the Kubernetes project are written** --
     `kubectl`, `kustomize`, etc

  3. it enables **composition of different types of tools written in different languages**
      * any modern language can manipulate yaml / json structures, no need to adopt `go`

  4. it **supports static analysis and validation**
      * develop tools and processes to perform validation and linting

  5. it **supports programmatic modification**
      * develop CLIs and UIs for working with configuration rather than using `vim`

  6. it **supports customizing generated Resources** so the Templates don't need to be modified
      * artifacts generated from Templates or DSLs may be modified directly, and then merged
        when they are regenerated to keep the modifications.

  7. it **supports display in UI and tools** which use either OpenAPI or the yaml/json directly.

### **Q: Isn't writing yaml hard?**

A: `kpt` offers a collection of utilities which enable working with configuration
   programmatically to simplify the experience.  Using `vi` to edit yaml should be
   necessary only for bootstrapping, and the common cases should use [setters](docs/cfg/set.md)
   or [functions](docs/fn/run.md) to generate or modify yaml configuration.

### **Q: I really like DSL / Templating solution X.  Can I use it with `kpt`?**

A: Yes. `kpt` supports plugging in solutions which generate or manipulate configuration, e.g. from
   DSLs and Templates.  This may be performed using [functions](docs/fn/run.md).  The generated
   output may be modified directly, and merged when regenerated.

### **Q: I want to write high-level abstractions like CRDs, but on the client-side.  Can I do this with `kpt`?**

A: Yes.  `kpt`'s architecture facilitates the developing programs which may generate or modify
   configuration.  Multiple programs may be composed together.  See [functions](docs/fn/run.md).

### **Q: How do I roll out changes throughout my organization using `kpt`?**

A: This can be done one of several ways, including: 1) using semantic versioning or release
   channels with [functions](docs/fn/run.md), or 2) [updating](docs/pkg/update.md) packages.

### **Q: Is there a container image that contains kpt?**

A: Yes. [gcr.io/kpt-dev/kpt](Dockerfile) contains the `kpt` and `kustomize` binaries.

## Community

**We'd love to hear from you!**

* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

---------------------

[demo]: https://storage.googleapis.com/kpt-dev/docs/overview-readme.gif "kpt"
