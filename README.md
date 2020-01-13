# KPT

Git based package management and toolchain for Kubernetes Resource Configuration.

- Publish, Consume and Update packages of Kubernetes Resource Configuration.
- Develop and Update Configuration programmatically.
- Filter and Display Configuration packages.
- Apply Configuration to clusters.

`kpt` combines package management commands with upstream Kubernetes tools to provide a complete
toolchain for building platforms for Kubernetes Resources.

![alt text][demo]

## Installation

Binaries:

- [darwin](https://storage.cloud.google.com/kpt-dev/kpt.master_darwin_amd64)
- [linux](https://storage.cloud.google.com/kpt-dev/kpt.master_linux_amd64)
- [windows](https://storage.cloud.google.com/kpt-dev/kpt.master_windows_amd64)

Source:

    GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt

### [Documentation](docs/README.md)

See the [docs](docs/README.md) for more information on how to use `kpt`.

## FAQ

### **Q: How is `kpt` different from other solutions?**

A: Rather than developing configuration by expressing it as code, `kpt` develops packages
   using the API native format -- i.e. as json or yaml objects adhering to the API schema.

### **Q: Why Resource configuration rather than Templates or DSLs?**  

A: Using Resource configuration provides a number of desirable properties:

  1. it clearly **represents the intended state** of the infrastructure -- no for loops, http calls,
    etc to interpret

  2. it **works directly tools developed by the Kubernetes project** -- `kubectl`, `kustomize`, etc

  3. it enables **composition of a variety of tools written in different languages**
      * any modern language can manipulate yaml / json structures, no need to adopt `go`

  4. it **supports static analysis**
      * develop tools and processes to perform validation and linting

  5. it enables package to be **modified programmatically**
      * develop CLIs and UIs for working with configuration rather than using `vim`

### **Q: Isn't writing yaml hard?**

A: `kpt` offers a collection of utilities which enable working with configuration
   programmatically to simply the experience.

### **Q: I really like DSL / Templating solution X.  Can I use it with `kpt`?**

A: `kpt` supports plugging in solutions which generate or manipulate configuration such
   as DSLs and Templates as [functions](docs/functions).

### **Q: I need to be able to write custom logic and develop high-level abstractions.  How can I do this with `kpt`?**

A: `kpt`'s architecture facilitates the development of customization and abstractions using
   a variety of techniques including [setters](docs/config/set.md) and [functions](docs/functions).

### **Q: How do I roll out changes throughout my organization using `kpt`?**

A: This can be done one of several ways, including: 1) using semantic versioning or release
   channels with [functions](docs/functions), or 2) [updating](docs/pkg/update.md) packages.

### **Q: Is there a container image that contains kpt?**

A: Yes. [gcr.io/kpt-dev/kpt](Dockerfile) contains the `kpt` and `kustomize` binaries.

## Community

**We'd love to hear from you!**

* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

---------------------

[demo]: docs/gifs/overview-readme.gif "Five Minute Demo"
