## cfg

Examine and modify configuration files

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg.cast" speed="1" theme="solarized-dark" cols="60" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    kpt tutorial cfg

[tutorial-script]

### Synopsis

Programmatically print and modify raw json or yaml Resource Configuration

| Command        | Description                                   |
|----------------|-----------------------------------------------|
| [annotate]     | set an annotation on one or more Resources    |
| [cat]          | print resources                               |
| [count]        | print resource counts                         |
| [create-setter]| create or modify a field setter               |
| [create-subst] | create or modify a field substitution         |
| [fmt]          | format configuration files                    |
| [grep]         | find resources by field value                 |
| [list-setters] | print available field setters                 |
| [set]          | set one or more field values                  |
| [tree]         | print resources as a tree                     |

**Data Flow**: local configuration or stdin -> kpt [cfg] -> local configuration or stdout

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

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
[create-subst]: create-subst.md
[fmt]: fmt.md
[grep]: grep.md
[list-setters]: list-setters.md
[set]: set.md
[tree]: tree.md
[fn]: ../fn/README.md
[pkg]: ../pkg/README.md

[tutorial-script]: ../gifs/cfg.sh
