## kpt

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/kpt.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    # run the tutorial from the cli
    kpt tutorial

[tutorial-script]

[pkg]: pkg/README.md
[cfg]: cfg/README.md
[fn]: fn/README.md
[tutorial-script]: gifs/kpt.sh

### Synopsis

kpt is a tool to help you manage, manipulate, customize, and apply Kubernetes resource
configuration files.  (And has a name short enough that you don't have to alias it to `k`).

kpt **package artifacts are composed of Resource configuration**, rather than code or templates.
However kpt does support **generating Resource configuration packages from arbitrary templates,
DSLs, programs,** etc.

| Command Group | Description                                       |
|---------------|---------------------------------------------------|
| [cfg]         | print and modify configuration files              |
| [pkg]         | fetch and update configuration packages           |
| [fn]          | generate, transform, validate configuration files |

---

#### [pkg] Package Management

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| git repository          | local files              |

**Data Flow**: git repo -> kpt [pkg] -> local files or stdout

Publish and share configuration as yaml or json stored in git.

- Publish blueprints and scaffolding for others to fetch and customize.
- Publish and version releases
- Fetch the blessed scaffolding for your new service
- Update your customized package by merging changes from upstream

---

#### [cfg] Configuration Management

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

**Data Flow**: local configuration or stdin -> kpt [cfg] -> local configuration or stdout

Examine and craft your Resources using the commandline.

- Display structured and condensed views of your Resources
- Filter and display Resources by constraints
- Set high-level knobs published by the package
- Define and expose new knobs to simplify routine modifications

---

#### [fn] Configuration Functions

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

**Data Flow**:  local configuration or stdin -> kpt [fn] (runs a docker container) -> local configuration or stdout

Run functional programs against Configuration to generate and modify Resources locally.

- Generate Resources from code, DSLs, templates, etc
- Apply cross-cutting changes to Resources
- Validate Resources

*`fn` is different from `cfg` in that it executes programs published as docker images, rather
than statically compiled into kpt.*

---

<!--

#### [svr] ApiServer Requests

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | apiserver                |
| apiserver               | stdout                   |

**Data Flow**: local configuration or stdin -> kpt [svr] -> apiserver (kubernetes cluster)

Push Resources to a cluster.

- Apply a package
- Wait until a package has been rolled out
- Diff local and remote state

-->

### Examples

    # get a package
    $ kpt pkg get https://github.com/GoogleContainerTools/\
      kpt.git/package-examples/helloworld-set@v0.1.0 helloworld
    fetching package /package-examples/helloworld-set from \
      git@github.com:GoogleContainerTools/kpt to helloworld

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

#### **Q: How is `kpt` different from other solutions?**

A: Rather than developing configuration by expressing it as code, `kpt` develops packages
   using the API native format -- i.e. as json or yaml objects adhering to the API schema.

#### **Q: Why Resource configuration as the artifact rather than Templates or DSLs?**  

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

#### **Q: Isn't writing yaml hard?**

A: `kpt` offers a collection of utilities which enable working with configuration
   programmatically to simplify the experience.  Using `vi` to edit yaml should be
   necessary only for bootstrapping, and the common cases should use [setters]
   or [functions] to generate or modify yaml configuration.

#### **Q: I really like DSL / Templating solution X.  Can I use it with `kpt`?**

A: Yes. `kpt` supports plugging in solutions which generate or manipulate configuration, e.g. from
   DSLs and Templates.  This may be performed using [functions].  The generated
   output may be modified directly, and merged when regenerated.

#### **Q: I want to write high-level abstractions like CRDs, but on the client-side.  Can I do this with `kpt`?**

A: Yes.  `kpt`'s architecture facilitates the developing programs which may generate or modify
   configuration.  Multiple programs may be composed together.  See [functions].

#### **Q: How do I roll out changes throughout my organization using `kpt`?**

A: This can be done one of several ways, including: 1) using semantic versioning or release
   channels with [functions], or 2) [updating](docs/pkg/update.md) packages.

#### **Q: Is there a container image that contains kpt?**

A: Yes. [gcr.io/kpt-dev/kpt] contains the `kpt` and `kustomize` binaries.

### Community

**We'd love to hear from you!**

* [gcr.io/kpt-dev/kpt]: Dockerfile)
* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

### 

[updating]: docs/pkg/update.md
[functions]: docs/fn/run.md
[setters]: docs/cfg/set.md
