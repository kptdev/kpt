## kpt

![alt text][demo]

### Synopsis

kpt is a Kubernetes platform toolkit for configuring Resources through configuration.

It includes tools developed as part of the Kubernetes project as well as additional commands
specific to `kpt`.

kpt packages are composed of Resource configuration, rather than code or templates.

The `kpt` command structure is as follows:

**Package Management: [pkg]**

Publish and share configuration as packages of yaml or json.

- Publish blueprints and scaffolding for others to fetch and customize.
- Publish and version releases
- Fetch the blessed scaffolding for your new service
- Update your customized package by merging changes from upstream

**Resource Configuration: [config]**

Examine and craft your package.

- Display structured and condensed views of your Resources
- Filter and display Resources by constraints
- Set high-level knobs published by the package
- Define and expose new knobs to simplify routine modifications

**Configuration Functions: [functions]**

Mixin public and custom programs which dynamically configure Resources on the client-side.

- Generate Resources from code, DSLs, templates, etc
- Apply cross-cutting changes to Resources
- Validate Resources

**Cluster Requests: [http]**

Push Resources to a cluster.

- Apply a package
- Wait until a package has been rolled out
- Diff local and remote state

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