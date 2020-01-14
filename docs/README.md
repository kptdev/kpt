## kpt

![alt text][demo]

### Synopsis

kpt is a Kubernetes platform toolkit targeted at developing and configuring Resource packages.

It includes tools developed as part of the Kubernetes project as well as additional commands
specific to `kpt`.

kpt packages are composed of Resource configuration, rather than code or templates.

The `kpt` command structure is as follows:

**Package Management: [pkg]**

- updating and syncing packages of Resource configuration from remote sources
- `get`, `update` and `diff` packages

**Resource Configuration: [config]**

- viewing and modifying Resource configuration
- `set` fields, print `tree` structure

**Configuration Functions: [functions]**

- generating, transforming and validating Resource configuration
- `run` functional-images locally against packages

**Cluster Requests: [http]**

- making requests to the Kubernetes control-plane
- `apply` and `diff` packages against clusters

------

To install shell completion for `kpt` commands and flags (bash, fish and zsh), run:

    COMP_INSTALL=1 kpt

To uninstall shell completion for kpt run:

    COMP_UNINSTALL=1 kpt

`kpt` invokes itself as its own completion command, which is registered with known shells
(e.g. .bashrc, .bash_profile, etc).

    complete -C /Users/USER/go/bin/kpt kpt

### 

[demo]: gifs/overview-readme.gif "Five Minute Demo"
[pkg]: pkg
[config]: config
[functions]: functions
[http]: http