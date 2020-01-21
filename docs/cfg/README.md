## cfg

Programmatically modify and view Resource configuration

![alt text][tutorial]

    kpt tutorial cfg

[tutorial-script]

### Synopsis

Programmatically print and modify raw json or yaml Resource Configuration

| Command        | Description                                   |
|----------------|-----------------------------------------------|
| [annotate]     | set `metadata.annotation`s on Resources       |
| [cat]          | print Resources in a package                  |
| [count]        | print Resource counts by type                 |
| [create-setter]| create or modify a custom field-setter        |
| [fmt]          | format Resource yaml                          |
| [grep]         | filter Resources configuration                |
| [list-setters] | list setters                                  |
| [set]          | set one or more fields programmatically       |
| [tree]         | print Resources using a tree structure        |

### Examples

    # print the raw package contents
    $ kpt cfg cat helloworld

    # print the package using tree based structure
    $ kpt cfg tree helloworld --name --image --replicas
    helloworld
    ├── [deploy.yaml]  Deployment helloworld-gke
    │   ├── spec.replicas: 5
    │   └── spec.template.spec.containers
    │       └── 0
    │           ├── name: helloworld-gke
    │           └── image: gcr.io/kpt-dev/helloworld-gke:0.1.0
    └── [service.yaml]  Service helloworld-gke

    # only print Services
    $ kpt cfg grep "kind=Service" helloworld | kpt cfg tree --name --image --replicas
    .
    └── [service.yaml]  Service helloworld-gke

    #  list available setters
    $ kpt cfg list-setters helloworld replicas
        NAME          DESCRIPTION        VALUE    TYPE     COUNT   SETBY
      replicas   'helloworld replicas'   5       integer   1

    # set a high-level knob
    $ kpt cfg set helloworld replicas 3
    set 1 fields

### Also See Command Groups

[fn], [pkg]

###

[annotate]: annotate.md
[cat]: cat.md
[count]: count.md
[create-setter]: create-setter.md
[fmt]: fmt.md
[grep]: grep.md
[list-setters]: list-setters.md
[set]: set.md
[tree]: tree.md

[fn]: ../fn/README.md
[pkg]: ../pkg/README.md

[tutorial]: https://storage.googleapis.com/kpt-dev/docs/cfg.gif "kpt cfg"
[tutorial-script]: ../../gifs/cfg.sh
