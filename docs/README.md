## kpt

  Git based configuration package manager.

#### Installation

    go install -v sigs.k8s.io/kustomize/kustomize/v3
    go install -v github.com/GoogleContainerTools/kpt

#### Commands

- [get](commands/get.md) -- fetch a package from git and write it to a local directory

      kpt help get # in-command help

      kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb@v0.1.0 my-cockroachdb
      kustomize config tree my-cockroachdb --name --replicas --image

      my-cockroachdb
      ├── [cockroachdb-statefulset.yaml]  Service cockroachdb
      ├── [cockroachdb-statefulset.yaml]  StatefulSet cockroachdb
      │   ├── spec.replicas: 3
      │   └── spec.template.spec.containers
      │       └── 0
      │           ├── name: cockroachdb
      │           └── image: cockroachdb/cockroach:v1.1.0
      ├── [cockroachdb-statefulset.yaml]  PodDisruptionBudget cockroachdb-budget
      └── [cockroachdb-statefulset.yaml]  Service cockroachdb-public

- [diff](commands/diff.md) -- display a diff between the local package copy and the upstream version

      kpt help diff # in-command help

      sed -i -e 's/replicas: 3/replicas: 5/g' my-cockroachdb/cockroachdb-statefulset.yaml
      kpt diff my-cockroachdb

      diff ...
      <   replicas: 5
      ---
      >   replicas: 3

- [update](commands/update.md) -- pull upstream package changes

      kpt help update # in-command help

      # commiting to git is required before update
      git add . && git commit -m 'updates'
      kpt update my-cockroachdb@v0.2.0

- [sync](commands/sync.md) -- manage a collection of packages using a manifest

      kpt help sync # in-command help

      kpt init . # init a new package
      kpt sync set https://github.com/GoogleContainerTools/kpt.git/package-examples/hello-world@v0.1.0 \
         hello-world # add a dependency
      kpt sync ./ # sync the dependencies 

      # print the package file
      cat Kptfile

      apiVersion: kpt.dev/v1alpha1
      kind: Kptfile
      dependencies:
      - name: hello-world
        git:
          repo: "https://github.com/GoogleContainerTools/kpt"
          directory: "package-examples/hello-world"
          ref: "v0.1.0"


- [desc](commands/desc.md) -- show the upstream metadata for one or more packages

      kpt help desc # in-command help

      kpt desc my-cockroachdb

       PACKAGE NAME         DIR                         REMOTE                       REMOTE PATH        REMOTE REF   REMOTE COMMIT  
      my-cockroachdb   my-cockroachdb   https://github.com/kubernetes/examples   /staging/cockroachdb   master       a32bf5c        

- [man](commands/man.md) -- render the README.md from a package if possible (requires man2md README format)

      kpt help man # in-command help

      kpt man my-cockroachdb

- [init](commands/init.md) -- initialize a new package with a README.md (man2md format) and empty Kptfile
  (optional)

      mkdir my-new-package
      kpt init my-new-package/

      tree my-new-package/
      my-new-package/
      ├── Kptfile
      └── README.md

#### Tutorials

- [fetch-a-package](tutorials/fetch-a-package.md)
- [update-a-local-package](tutorials/update-a-local-package.md)
- [publish-a-package](tutorials/publish-a-package.md)

#### Design

1. **Packages are composed of Resource configuration** (rather than DSLs, templates, etc)
    * May also contain supplemental non-Resource artifacts (e.g. README.md, arbitrary other files).

2.  **Any existing git subdirectory containing Resource configuration** may be used as a package.
    * Nothing besides a git directory containing Resource configuration is required.
    * e.g. the [examples repo](https://github.com/kubernetes/examples/staging/cockroachdb) may
      be used as a package:

          # fetch the examples cockroachdb directory as a package
          kpt get https://github.com/kubernetes/examples/staging/cockroachdb my-cockroachdb

3. **Packages should use git references for versioning**.
    * Package authors should use semantic versioning when publishing packages.

          # fetch the examples cockroachdb directory as a package
          kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb@v0.1.0 my-cockroachdb

4. **Packages may be modified or customized in place**.
    * It is possible to directly modify the fetched package.
    * Tools may set or change fields.
    * [Kustomize functions](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/commands/run-fns.md)
      may also be applied to the local copy of the package.

          export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true

          kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb my-cockroachdb
          kustomize config set my-cockroachdb/ replicas 5

5. **The same package may be fetched multiple times** to separate locations.
    * Each instance may be modified and updated independently of the others.

          export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true

          # fetch an instance of a java package
          kpt get https://github.com/GoogleContainerTools/kpt/examples/java my-java-1
          kustomize config set my-java-1/ image gcr.io/example/my-java-1:v3.0.0

          # fetch a second instance of a java package
          kpt get https://github.com/GoogleContainerTools/kpt/examples/java my-java-2
          kustomize config set my-java-2/ image gcr.io/example/my-java-2:v2.0.0

6. **Packages may pull upstream updates after they have been fetched and modified**.
    * Specify the target version to update to, and an (optional) update strategy for how to apply the
      upstream changes.

          export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true

          kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb my-cockroachdb
          kustomize config set my-cockroachdb/ replicas 5
          kpt update my-cockroachdb@v1.0.1 --strategy=resource-merge

#### Building Platforms, Solutions and High-Level Systems with kpt

kpt was developed to solve **configuration packaging** only -- and was designed to be composed
with other tools from the ecosystem in order to build higher-level systems and platforms.

As such, **kpt has a minimal feature set by design to maximize its utility as a building block
for platforms** and ability to be composed with other tools.

This sections provides a high-level overview of some of the ways in which kpt may be composed
with tools from the upstream Kubernetes project to build configuration and delivery solutions.

See [building-solutions](tutorials/building-solutions.md) for a more information on this topic.

1. **Packaging**

   Packaging covers how to bundle configuration for reuse.

   - Fetch -- get a bundle of Resource configuration
   - Update -- pull in upstream changes to Resource configuration
   - Publish -- publish a bundle of Resource configuration

2. **Development**

   Development covers how to create and modify configuration, and includes
   how to incorporate and unify opinions from an arbitrary number of sources.

   - Abstraction -- substitution, generation, injection, etc
   - Customization -- configuring blueprints, defining variants, etc
   - Validation -- policy enforcement, linting, etc

3. **Actuation**

   Actuation covers how to take configuration and actuate it by applying it.

   - Apply -- apply configuration to a cluster
   - Status -- waiting for changes to be fully rolled out
   - Prune -- deletion of Resources no longer appearing in the config

4. **Visibility** /  **Inspection**

   Visibility / Inspection covers how to visualize and understand packaged
   configuration, as well as applied Resources.

   - Search for Resources within a Package, Cluster or set of Clusters.
   - Visualize the relationship between Resources
   - Debug Resources

5. **Discovery**

   Discovery includes how to locate new packages, and examples.

   - Discover new publicly published packages from a market place or the web

#### Tools

| Category      | Example Tool           | Example Commands                                  |
|---------------|------------------------|---------------------------------------------------|
| Packaging     | `kpt`                  | `kpt get`, `kpt update`                           |
| Development   | `kustomize`            | `kustomize build`, `kustomize config run`         |
| Actuation     | `kubectl`, `kustomize` | `kubectl apply`, `kustomize status`               |
| Visibility    | `kustomize`, `kubectl` | `kustomize config grep`, `kustomize config tree`  |
| Discovery     | GitHub                 |                                                   |

#### FAQ

See [faq](tutorials/faq.md)

#### Templates and DSLs

Note: If the use of Templates or DSLs is strongly desired, they may be fully expanded into Resource
configuration to be used as a kpt package.  These artifacts used to generated Resource configuration
may be included in the package as supplements.

#### Env Vars

  COBRA_SILENCE_USAGE
  
    Set to true to silence printing the usage on error

  COBRA_STACK_TRACE_ON_ERRORS

    Set to true to print a stack trace on an error

  KPT_NO_PAGER_HELP

    Set to true to print the help to the console directly instead of through
    a pager (e.g. less)
