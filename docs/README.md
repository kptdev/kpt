## kpt

![alt text][demo]

### Synopsis

kpt is a Kubernetes platform toolkit.

It includes tools to package, customize and apply json or yaml configuration data.
This includes tools developed as part of the Kubernetes project as well as additional commands
specific to `kpt`.

kpt package artifacts are composed of Resource configuration, rather than code or templates,
but code or templates may be plugged into kpt to generate configuration.

#### Major `kpt` subcomponents

**Package Management: [pkg]**

Publish and share configuration as yaml or json stored in git.

- Publish blueprints and scaffolding for others to fetch and customize.
- Publish and version releases
- Fetch the blessed scaffolding for your new service
- Update your customized package by merging changes from upstream

**Configuration Management: [cfg]**

Examine and craft your Resources using the commandline.

- Display structured and condensed views of your Resources
- Filter and display Resources by constraints
- Set high-level knobs published by the package
- Define and expose new knobs to simplify routine modifications

**Configuration Functions: [fn]**

Run functional programs against Configuration to generate and modify Resources locally.

- Generate Resources from code, DSLs, templates, etc
- Apply cross-cutting changes to Resources
- Validate Resources

**ApiServer Requests: [svr]**

Push Resources to a cluster.

- Apply a package
- Wait until a package has been rolled out
- Diff local and remote state

### Examples

    # learn about kpt
    $ kpt help

    # get a package
    $ kpt pkg get https://github.com/GoogleContainerTools/\
      kpt.git/package-examples/helloworld-set@v0.1.0 helloworld
    fetching package /package-examples/helloworld-set from \
      git@github.com:GoogleContainerTools/kpt to helloworld

    # list setters
    $ kpt cfg list-setters helloworld
    NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY
    http-port   'helloworld port'         80      integer   3
    image-tag   'hello-world image tag'   0.1.0   string    1
    replicas    'helloworld replicas'     5       integer   1

    # set a value
    $ kpt cfg set helloworld replicas 3 --set-by pwittrock \
      --description '3 is good enough'
    set 1 fields

    # apply
    $ kpt svr apply -f helloworld
    deployment.apps/helloworld-gke created
    service/helloworld-gke created

### 

[demo]: https://storage.googleapis.com/kpt-dev/docs/overview-readme.gif "kpt"
[pkg]: pkg
[cfg]: cfg
[fn]: fn
[svr]: svr