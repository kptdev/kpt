## kpt

![alt text][demo]

[image-script](../gifs/kpt.sh)

### Synopsis

*kpt* is a Kubernetes platform toolkit.

- It includes tools to package, customize and apply json or yaml configuration data.
- It includes tools developed as part of the Kubernetes project as well as additional commands
  specific to *kpt*.

*kpt* package artifacts are composed of Resource configuration, rather than code or templates,
however *kpt* supports using code or templates as solutions to generate *kpt* package artifacts,
which may then be consumed by other tools as Resource configuration.

---

#### [pkg] Package Management

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| git repository          | local files              |

Flow: git repo -> kpt [pkg] -> local configuration or stdout

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

Flow: local configuration or stdin -> kpt [cfg] -> local configuration or stdout

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

Flow:  local configuration or stdin -> kpt [fn] (runs a docker container) -> local configuration or stdout

Run functional programs against Configuration to generate and modify Resources locally.

- Generate Resources from code, DSLs, templates, etc
- Apply cross-cutting changes to Resources
- Validate Resources

*`fn` is different from `cfg` in that it executes programs published as docker images, rather
than statically compiled into kpt.*

---

#### [svr] ApiServer Requests

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | apiserver                |
| apiserver               | stdout                   |

Flow: local configuration or stdin -> kpt [svr] -> apiserver (kubernetes cluster)

Push Resources to a cluster.

- Apply a package
- Wait until a package has been rolled out
- Diff local and remote state

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
    $ kpt svr apply -R -f helloworld
    deployment.apps/helloworld-gke created
    service/helloworld-gke created

    # learn about kpt
    $ kpt help

### 

[demo]: https://storage.googleapis.com/kpt-dev/docs/kpt.gif "kpt"
[pkg]: pkg/README.md
[cfg]: cfg/README.md
[fn]: fn/README.md
[svr]: svr/README.md