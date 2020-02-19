## kpt

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/kpt.cast" speed="1" theme="solarized-dark" cols="60" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    # run the tutorial from the cli
    kpt tutorial

[tutorial-script]

[pkg]: pkg/README.md
[cfg]: cfg/README.md
[fn]: fn/README.md
[tutorial-script]: gifs/kpt.sh

### Quickstart Guides

| Guides                  | Description                                                     |
|-------------------------|-----------------------------------------------------------------|
| [Consuming a Package]   | fetch a new copy of a package, writing it to a local directory  |
| [Publishing a Package]  | publish a new package for others to consume                     |

[Consuming a Package]: quick-start-guides/consumer-quick-start-guide.md
[Publishing a Package]: quick-start-guides/producer-quick-start-guide.md

---

### Synopsis

kpt is a toolkit to help you manage, examine, manipulate, customize, validate, and apply Kubernetes resource
configuration files, both manually and programmatically.  (And has a name short enough that you don't have to alias it to `k`).

A primary goal of kpt is to facilitate configuration reuse. The two primary sets of capabilities that are required to enable reuse are:
1. The ability to distribute/publish/share, compose, and update groups of configuration artifacts, commonly known as packages.
2. The ability to adapt them to your use cases, which we call customization.

In order to facilitate programmatic operations, kpt:
1. Relies upon git as the source of truth
2. Represents configuration as data, specifically Kubernetes resources serialized as YAML or JSON.

For compability with other arbitrary formats, kpt supports generating resource configuration data from templates,
configuration DSLs, and programs.

kpt functionality is subdivided into command groups, each of which operates on a particular set of entities, with a consistent command syntax and pattern of inputs and outputs.

| Command Group | Description                                                                     |
|---------------|---------------------------------------------------------------------------------|
| [pkg]         | fetch, update, and sync configuration files using git                           |
| [cfg]         | examine and modify configuration files                                          |
| [fn]          | generate, transform, validate configuration files using containerized functions |
| TODO          | reconcile configuration files with the live state                               |

---

#### [pkg] Package Management

Fetch, update, and sync configuration files using git.

- Fetch and customize blueprints published by others.
- Fetch the standard scaffolding for your new service.
- Update your customized package by merging changes from upstream.

**Data Flow**: git repo -> kpt [pkg] -> local files or stdout

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| git repository          | local files              |

---

#### [cfg] Configuration Management

Examine and modify configuration files.

- Display structured and condensed views of your resources
- Filter and display resources by constraints
- Set high-level knobs published by the package
- Define and expose new knobs to simplify routine modifications

**Data Flow**: local configuration or stdin -> kpt [cfg] -> local configuration or stdout

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

---

#### [fn] Configuration Functions

Generate, transform, validate configuration files using containerized functions.

- Generate resources from code, DSLs, templates, etc.
- Apply cross-cutting transformations to resources
- Validate resources

*`fn` is different from `cfg` in that it executes programs published as container images, rather
than statically compiled into kpt.*

**Data Flow**:  local configuration or stdin -> kpt [fn] (runs a container) -> local configuration or stdout

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

---

#### Coming soon: Live-state Management

Reconcile configuration files with the live state.

- Apply a package
- Preview changes before applying them
- Wait until a package has been fully reconciled with the live state
- Diff local configuration files and the live state

**Data Flow**: local configuration or stdin -> kpt TODO -> apiserver (Kubernetes cluster)

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | apiserver                |
| apiserver               | stdout                   |

---

### Examples

    # get a package
    $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld
    fetching package /package-examples/helloworld-set from \
      https://github.com/GoogleContainerTools/kpt to helloworld

    # list setters and set a value
    $ kpt cfg list-setters helloworld
    NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY
    http-port   'helloworld port'         80      integer   3
    image-tag   'hello-world image tag'   0.1.0   string    1
    replicas    'helloworld replicas'     5       integer   1

    $ kpt cfg set helloworld replicas 3 --set-by pwittrock  --description 'reason'
    set 1 fields

    # apply
    $ kubectl apply -R -f helloworld
    deployment.apps/helloworld-gke created
    service/helloworld-gke created

    # learn about kpt
    $ kpt help

### FAQ

#### **Q: What's with the name?**

A: `kpt` was inspired by `apt`, but with a Kubernetes focus.  We wanted to uphold the tradition
   of naming tools to start with `k`, and also be short enough that you don't have to alias it.
   It is pronounced "kept".

#### **Q: How is `kpt` different from other solutions?**

A: Rather than expressing configuration as code, `kpt` represents configuration packages as data, 
   in particular as YAML or JSON objects adhering to the 
   [Kubernetes resource model](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/resource-management.md).

#### **Q: Why resource configuration as the artifact rather than templates or configuration DSLs?**  

A: As explained in [Declarative application management in Kubernetes](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md),
   using resource configuration provides a number of desirable properties:

  1. it clearly **represents the intended state** of the infrastructure -- no for loops, http calls,
    etc to interpret

  2. it **aligns with how tools developed by the Kubernetes project are written** --
     `kubectl`, `kustomize`, etc

  3. it enables **composition of different types of tools written in different languages**
      * any modern language can manipulate YAML / JSON structures, no need to adopt `go`

  4. it **supports static analysis and validation**
      * develop tools and processes to perform validation and linting

  5. it **supports programmatic modification**
      * develop CLIs and UIs for working with configuration rather than using `vim`

  6. it **supports customizing generated resources** so the templates don't need to be modified
      * artifacts generated from templates or DSLs may be modified directly, and then merged
        when they are regenerated to keep the modifications.

  7. it **supports display in UI and tools** which use either OpenAPI or the YAML/JSON directly.

#### **Q: Isn't writing YAML hard?**

A: `kpt` offers a collection of utilities which enable working with configuration
   programmatically to simplify the experience.  Using `vi` to edit YAML should be
   necessary only for bootstrapping, and the common cases should use [setters]
   or [functions] to generate or modify YAML configuration.

#### **Q: I really like DSL / templating solution X.  Can I use it with `kpt`?**

A: Yes. `kpt` supports plugging in solutions which generate or manipulate configuration, e.g. from
   DSLs and templates.  This may be performed using [functions].  The generated
   output may be modified directly, and merged when regenerated.

#### **Q: I want to write high-level abstractions like CRDs, but on the client-side.  Can I do this with `kpt`?**

A: Yes.  `kpt`'s architecture facilitates the developing programs which may generate or modify
   configuration.  Multiple programs may be composed together.  See [functions].

#### **Q: How do I roll out changes throughout my organization using `kpt`?**

A: This can be done one of several ways, including: 1) using semantic versioning or release
   channels with [functions], or 2) [updating] packages.

#### **Q: Is there a container image that contains kpt?**

A: Yes. [gcr.io/kpt-dev/kpt] contains the `kpt` binary.

### Community

**We'd love to hear from you!**

* [gcr.io/kpt-dev/kpt]
* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

### 

[updating]: pkg/update.md
[functions]: fn/run.md
[setters]: cfg/set.md
[gcr.io/kpt-dev/kpt]: https://gcr.io/kpt-dev/kpt

