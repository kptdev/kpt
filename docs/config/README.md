## config

View and Modify Resource Configuration.

![alt text][demo]

### Synopsis

`config` contains commands for building and understanding Resource configuration packages.

### Primary Commands

**[tree](tree.md), [count](count.md), [cat](cat.md)**:
- print package contents as a tree, aggregate counts, or raw configuration

**[set](set.md), [list-setters](list-setters.md), [create-setter](create-setter.md)**:
- modify Resources using high-level knobs with `set`
- list high-level knobs
- create new high-level knobs

### Additional Commands

**[fmt](fmt.md)**:
- format configuration by sorting fields 

**[grep](grep.md)**:
- search for Resource matching filters

**[merge](merge.md), [merge3](merge3.md)**:
- merge one collection of Resources into another by GVK + namespace + name

### Examples

    # print the raw package contents
    $ kpt config cat helloworld

    # print the package using tree based structure
    $ kpt config tree helloworld --name --image --replicas
    helloworld
    ├── [deploy.yaml]  Deployment helloworld-gke
    │   ├── spec.replicas: 5
    │   └── spec.template.spec.containers
    │       └── 0
    │           ├── name: helloworld-gke
    │           └── image: gcr.io/kpt-dev/helloworld-gke:0.1.0
    └── [service.yaml]  Service helloworld-gke

    # only print Services
    $ kpt config grep "kind=Service" helloworld | kpt config tree --name --image --replicas
    .
    └── [service.yaml]  Service helloworld-gke

    #  list available setters
    $ kpt config list-setters helloworld replicas
        NAME          DESCRIPTION        VALUE    TYPE     COUNT   SETBY
      replicas   'helloworld replicas'   5       integer   1

    # set a high-level knob
    $ kpt config set helloworld replicas 3
    set 1 fields

### 

[demo]: ../gifs/config.gif "kpt config overview"
