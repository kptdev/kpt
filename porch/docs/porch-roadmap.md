# Porch Roadmap

Last updated: February 9th, 2023

This document outlines next steps for Porch in several areas. This is a living
document outlining future direction and work in different Porch subsystems.

## API Changes

* Expression kpt function type 'signature', including:
  * types of resources the function acts on
  * schema of the function config
  * types of resources the function _produces_ (if the function introduces new
    resources into the configuration package)
* Add `Package` resource to encapsulate all `PackageRevision`s of the same
  package, link to the latest `PackageRevision`. Possibly the `Package` resource
  can allow mutation of the package or its contents, and automatically create
  new (_Draft_) `PackageRevision` resources on mutations.
* Implement support for API-level filtering (field and label selectors) for all
  `list` operations.
* Make sure that all errors translate to the appropriate API-level HTTP status
  with clear, actionable messages.
* Cross-namespace cloning to make it possible to better leverage Kubernetes RBAC
  for controlling access to repositories.
* Move long-running operations to asynchronous operations by controllers rather than
  synchronous operations by the aggregated apiserver. This means exposing results
  in the status object of resources rather than return errors.
* Support non-KRM content as part of `ResourceList` to allow lossless package
  transformations, or compensate for lack of this support in general by enabling
  in Porch partial package revision `pull` and `push`.

## Repository Management

* Support for specifying repository-wide, default _upstream_ repository which
  would become the default upstream repository for cloned packages
* Repository-wide guardrails - functions registered with the repository that
  then are evaluated on packages in that repository whenever those packages
  change.
* Support updating repository registration, for example when `Repository`
  resource is modified to point to a different repository, or even a different
  type of a repository (Git --> OCI).
* Support read only repositories. Porch will allow package discovery in a read
  only repository but attempt to create/modify package in read only repository
  will result in an error. Consider supporting via RBAC so different principals
  can have different level of access to the same repository.
* Implement repository cache eviction policy
* Support `ObjectMeta.GenerateName` for `PackageRevision` creation. Currently
  package names are computed as <repository>-<sha>. Ideally Porch would accept
  name prefix (constrained for example to the last segment of the package name)
  as `GenerateName` value, for example `istions` and Porch would append the SHA.
  This will require creating an inverse mapping from package name to its owning
  repository. Currently the inverse mapping is encoded in the package name,
  whose format is `<repository name>-<package name hash>`.
* Error-resilient repository ingestion (some erroneous packages in the
  repository should not prevent repository from successfully loading).
* Make the background repository routine a controller to leverage functionality
  in the common controller libraries, such as maintaining reliable `Watch`
  connection (`background.go`).
* Enable OCI repositories with heterogeneous contents (containing both
  `Package` and `Function` resources)?
* Improve synchronization of repository actions to avoid simultaneous repository
  fetches and repository access that requires synchronization.

### Git

* Support authentication for cloning packages from unregistered repositories.
* Porch will need to store more information associated with a package revision,
  for example:
  * information about the package's error status (to populate the `status`
    section of the PackageRevision API resource))
  * rationale for rejection proposal to publish a package

  We will need to utilize Git repository or some auxiliary storage (git is
  preferred because it also propagates information across Porch instances)
  and we can consider:
  * storing the information in a HEAD draft commit. When draft is updated,
    Porch would drop this HEAD commit, stack more mutation commits and then
    add a new HEAD commit with additional meta information
  * using separate 'meta' branches
  * Git notes (though these are attached to object )
* Implement appropriate caching when cloning packages from unregistered
  repositories (currently, those repositories are fetched and immediately
  deleted))

## OCI

* Complete the OCI support (current implementation is partial, missing package
  lifecycle support, cloning from OCI repository is not supported, etc.)
* Support for authentication methods as required by integration with specific
  OCI repository providers. Currently Porch authenticates as the workload
  identity GCP service account which works well with Google Container Registry
  and Artifact Registry; different methods will likely be needed for other
  providers.

## Engine

* Create a unified representation of a package and its contents in the system.
  Porch Engine currently stores package contents as `map[string]string` (file
  name --> file contents, see `PackageResources` type) and kpt intrinsic
  algorithms work with `kyaml filesys.FileSys` interface, necessitating
  translation. Ideal representation would help minimize the need to not only
  translate the representation at the macro level but also reduce need for
  repeated parsing and serialization of YAML.
* Support package contents that are not text
* Revisit package update mutation to avoid using local file system and integrate
  better with CaD library (`updatePackageMutation.Apply`).
* Migrate the PackageRevision resource to a CRD rather than using the
  aggregated APIServer.

## Package Lifecycle

* Ensure that all operations can be performed only on package revision in the
  appropriate lifecycle state (example: only _Published_ packages can be cloned,
  deployed, etc.)
* Support detection of new version of upstream package and downstream package
  update
* Support sub-packages
* Handling of merge conflicts, assistance with manual conflict resolution
* Bulk package operations (bulk upgrade from updated upstream package)
* Permission model to enable admins more fine-grained control over repository
  access than what it supported by underlying Git or other providers

## CLI Integration

* Support for registering repository-wide guardrails - mutators or validators.
* Support for updating repository registration (currently only `register`,
  `unregister`, and `get` are implemented)
* On repository unregistration, the CLI can check all other registered
  repositories and suggest to the user to keep or delete the secret containing
  credentials depending on whether other repository registrations use it or
  the last one is being deleted; today, a `--keep-auth-secret` flag is used.
* Registration of _function_ repositories (only _package_ repositories are
  currently supported in the CLI).
* Function discovery via in registered function repositories.
* Consider revising the structure of the `kpt alpha` command groups. `kubectl`
  organizes the groups by action, for example `kpt get <resource>` whereas
  `kpt` by resource: `kpt alpha rpkg get`. Consider `kpt alpha get rpkg` or
  `kpt alpha get repo` for consistency with `kubectl` experience.
* Implement richer support for referencing to packages by URLs with proper
  parsing, reduce number of command line flags required. For example, support:
  `https://github.com/org/repo.git/packge/patch@reference`; Make the parsing
  code reusable with the rest of kpt CLI.

## Testing

* Enable the e2e tests to run against a specified Git repository
* Accept Git test image as an argument to avoid requiring the Porch server
  image and Git test server image to share the same tag
  (see `InferGitServerImage` function and `suite.go` file)
* Set up infrastructure for better testing of the controllers.

## Deployment and integration with syncers
* Support bulk management of variants of packages based on deployment targets.
* Rollout engine for progressive rollout of packages into clusters.



