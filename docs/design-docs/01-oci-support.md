# OCI Support

* Author(s): Louis Dejardin, @loudej
* Approver: \<kpt-maintainer\>

>    Every feature will need design sign off an PR approval from a core
>    maintainer.  If you have not got in touch with anyone yet, you can leave
>    this blank and we will try to line someone up for you.

## Why

Systems that deal with packaging software or bundling configuration often have
an atomic, versionable artifact. This artifact can exist as a source of
truth independant from the source controlled content from which it was built.

It is also very common for those packages to have an associated feed or repository
which can receive those packages as they are published, and make them available for 
download as needed. In many companies, using `git` as the as the repository and source
of truth for production configuration comes with challenges.

This design document proposes to add `OCI` as an alternative to `git` for publishing and
distributing Kpt config packages. As a packaging format, it is well understood and documented.
As a repository format, it leverages existing container registries for pushing and pulling
config as image content. For security and production configuration management, if a 
company has practices for managing Docker container images in private registries, then
the same practices and security model can be applied to config package images in private
registries.

https://github.com/GoogleContainerTools/kpt/issues/2300

## Design

### Design Assumptions

The first stage of OCI support comes from adding support for `oci` in the places
where `git` appears today.

An image tag is used in the same way a git branch or tag would be used.

An image digest is used in the same way a git commit would be used.

The scope of a single image is one root package, with any number of optional sub-packages.

The structure of the image is a single tar layer. The root `Kptfile` is in the base directory from the tar layer's point of view. It contains only the Kpt package files, no entrypoint or executables.

A package image should not be confused with a container image. Container images are executable by software, like `Docker`, and package images are purely configuration data.

### Config chages

The `Kptfile` structures for `upstream` and `upstreamLock` have `oci` in addition to `git` properties. The `type` property also has the string `oci` added as an accepted value.

```yaml
upstream:
  type: oci
  oci:
    image: 'IMAGE:TAG'
upstreamLock:
  type: oci
  oci:
    image: 'IMAGE:DIGEST'
```

New verions of `kpt` will support existing `Kptfile`. The file structures and `git` functionality is unchanged.

Existing versions of `kpt` will support `Kptfile` with `upstream` based on `git` for the same reason. The structure and meaning of existing fields is not changed.

Existing versions of `kpt` will not support `Kptfile` with `upstream` based on `oci`. The `type` value, and missing `git` information will fail validation. The `kpt` binary used will need to be upgraded.

### Command changes

### `kpt pkg get`

The argument that determines upstream today is parsed into `repo`, `ref`, and `path`, and is implicitly a `git` location.

To support `oci`, it will be necessary to extract different values in a way that's unambiguous. Unfortunately, OCI image names have no Uri prefix, and are indistinguishable from a valid path or file name.

To solve this, using [Helm](https://helm.sh/docs/topics/registries/#other-subcommands) as an example, the prefix `oci://` can be used. This ensures that selecting `oci` protocol isn't accidental, and it won't collide with other location formats that may be added.

```shell
# clone package as new folder
kpt pkg get oci://us-docker.pkg.dev/the-project-id/the-repo-name/the-package:v3 my-package
```

Because OCI image reference already has a convention for `image:tag` references, using `:v3` should be used instead of `@v3` for version. It will be more intuitive how it relates to the registry, and easier to cut and paste values.

### `kpt pkg get` sub-packages

It is possible to use `kpt pkg get` to add sub-packages to a target location.

Syntax for a sub-package target location is unchanged, it's a normal filesystem path.

Syntax for an OCI sub-package source location requires the ability to tell when an image name ends and a sub-package path inside that image begins. In `git` this requires an explicit `.git` extension at the transition, and in `.oci` this requires double slash.

```shell
# clone sub-package as new sub-folder
kpt pkg get oci://us-docker.pkg.dev/the-project-id/the-repo-name/the-package//simple/example:v3 my-package/simple/my-example
```

### `kpt pkg update`

The command for update is not changed, but when the `upstream` is `oci` then the `@VERSION` is used to change the `upstream` image's `tag` or `digest` value.

To update to an image tag, `kpt pkg update @v14` and `kpt pkg update DIR@v14` will assign the `:v14` tag onto the upstream image.

```yaml
upstream:
  type: oci
  oci:
    image: us-docker.pkg.dev/the-project-id/the-repo-name/the-package:v14
```

To update to an upstream digest, `kpt pkg update @sha256:{SHA256_HEX}` and `kpt pkg update DIR@sha256:{SHA256_HEX}` will assign `@sha256:{SHA256_HEX}` as the new upstream image digest.

```yaml
upstream:
  type: oci
  oci:
    image: us-docker.pkg.dev/the-project-id/the-repo-name/the-package@sha256:8815143a333cb9d2cb341f10b984b22f3b8a99fe
```

Calling `kpt pkg update` and `kpt pkg update DIR` will perform an update without changing the upstream image name.

At that point, if the `upstream` is an `image:tag` that is to discover the current `image:digest` for tag, otherwise the `upstream` value for `image:digest` is used. In either case, the `upstreamLock` is changed to point at that new `image:digest`. 

The package contents of the old and new `upstreamLock` image digest are fetched to temp folders, and are the basis of the 3-way merge to update the target package.

### `kpt pkg diff`

The `kpt pkg diff` command is identical to `kpt pkg update` in the way that `[PKG_PATH@VERSION]` argument is mapped to OCI concepts.

### Command additions

Although it is possible to create and push an OCI image using a combination of commands like `tar` and `gcrane`, that
doesn't provide a very complete end to end experience. Because kpt would already be built with the same OCI go module used 
by `gcrane`, it is not difficult to support additional commands to move pull and push package contents from local folders
to remote images and back.

### ` kpt pkg pull`

```
Usage: kpt pkg pull oci://{IMAGE[:TAG|@sha256:DIGEST]} [DIR] 
  DIR                         Destination folder for image contents. Default folder name is the last part of the IMAGE path.
  IMAGE[:TAG|@sha256:DIGEST]  Name of image to pull contents from, with optional TAG or DIGEST. Default TAG is `Latest`
```

This command is the reverse of push. An image can be pulled from a repository to a local folder, modified, and pushed
back to the same location, same location with different TAG, or entirely different location.

The target DIR is optional, following the conventions of `kpg pkg get`, and will default to the final image/path segment.

`kpt pkg pull` works on git uri as well. This may be used, for example, to mirror a set of known blueprints into a private 
oci registry.

### ` kpt pkg push`

```
Usage: kpt pkg push [DIR@VERSION] [--origin oci://{IMAGE[:TAG]}] [--increment]
  DIR@VERSION     Folder containing package root Kptfile. Default is current directory.
                  Optional @VERSION changes tag or branch to push onto. Default is most recently pulled or pushed tag.
  --origin        Name of image to push contents onto, with optional TAG to assign to resulting commit.
                  Default is to use most recently pulled/pushed image. Required if Kptfile does not have an origin.
  --increment     Increase the version by 1 while pushing. Default is to leave the origin's TAG or DIR@VERSION unchanged.
                  The Kptfile's image TAG is also updated to the new value. 
```

This command will `tar` the contents of the package into a single image layer, and push it into the OCI repository. For
Google Artifact Registry and Google Container Registry, the current `gcloud auth` SSO credentials are used.

The simplest form of the command is `kpt pkg push` or `kpt pkg push DIR` which will push the current contents back to
the IMAGE:TAG location that was saved when `kpt pkg pull` was run.

The synxax `kpt pkg push @VERSION` or `kpt pkg push DIR@VERSION` will push back to the image location it came from, but with a new TAG name or version. Examples are `kpt pkt push @draft` or `kpt pkg push @v4`

If the Kptfile was not obtained by `kpt pkg pull` - for example it's a new package from `kpt pkg init` or `kpt pkg get` - then
the first `kpt pkg push` will require an `--origin IMAGE:TAG` option to provide the target location. It is only necessary on the first
call.

Finally, if the IMAGE's TAG value is a valid version number, the `--increment` switch can be used to add 1 to the current value before pushing.

In the simplest case a `v1` is changed to `v2`, and `1` is changed to `2`, but any TAG that is a valid semver (with optional leading 'v') will have the smallest part of the number incremented. So `v1.0` becomes `v1.1`, `v1.0.0` becomes `v1.0.1`, and `v4.1.9-alpha` becomes `v4.1.10-alpha`

#### Comparison of `pkg get` and `pkg pull`

Starting with a simple root package, and an orange variant with root as the upstream:

```
-- root
  \-- orange {upstream: root}
```

The purpose of `kpt get` is to create a new leaf node. This is done by creating the initial copy of the new leaf package in a
local folder. This has the side-effects of altering the kptfile name, the upstream values to point at the source, and makes appropriate 
changes to sub-package metadata.

As an example, after running `kpt pkg get scheme://repo/root green` and `kpt pkg get scheme://repo/orange blue` the `green` and 
`blue` local folder packages are appended to the inheritance tree like this:

```
-- root
  \-- orange {upstream: root}
  | `-- blue {upstream: orange}  ** working copy in ./blue **
  \-- green {upstream: root}     ** working copy in ./green **
```

By comparison, `kpt pkg pull` does not create a new package node or identity - it only extracts a copy of existing package
contents to a working directory. In this example, if the user additionally ran `kpt pkg pull scheme://repo/root root` and 
`kpt pkg pull scheme://repo/orange orange` the overall state would be this:

```
-- root                          ** working copy in ./root **
  \-- orange {upstream: root}    ** working copy in ./orange **
  | `-- blue {upstream: orange}  ** working copy in ./blue **
  \-- green {upstream: root}     ** working copy in ./green **
```

### Alternatives to push/pull

There are several ways that pull and push could appear as commands. Those two names are very conventional, but 
alternatives to consider could be:

### `kpt pkg copy`

```
Usage: kpt pkg copy {SOURCE} {DEST}
  SOURCE  Package source location: a local DIR, or `oci://` image, or git repo and path
  DEST    Package destination: a local DIR, or `oci://` image.
```

Puts a copy of the SOURCE package at the DEST location. The package contents would be entirely unchanged by this operation (unlike `kpt pkg get`).

To pull from remote image to local folder:

```
kpt pkg copy \
  oci://us-docker.pkg.dev/the-project-id/the-repo-name/the-package:v14 \
  the-package
```

To push from local folder to new remote image tag:

```
kpt pkg copy \
  the-package \
  oci://us-docker.pkg.dev/the-project-id/the-repo-name/the-package:v15
```
 
To copy a package image from one OCI repo to another:

```
kpt pkg copy \
  oci://us-docker.pkg.dev/the-project-id/dev-blueprints/the-package:v25 \
  oci://us-docker.pkg.dev/the-project-id/prod-blueprints/the-package:v25
```

To copy a package from a git location to an OCI repo:

```
kpt pkg copy \
  https://github.com/GoogleCloudPlatform/blueprints.git/catalog/gke@main \
  oci://us-docker.pkg.dev/the-project-id/gcp-catalog/gke:latest
```

## User Guide

### Creating a package respository

Before kpt packages can be pushed and pulled as OCI images, a suitable repository 
must be created. Google Artifact Registry and Google Container Registry are both
excellent choices.

```shell
# Choose names and locations
LOCATION="us"
PROJECT_ID="kpt-demo-73823"
REPOSITORY_NAME="blueprints"

# Base name for any images in this repository
REPOSITORY="${LOCATION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}"

# Create the repository
gcloud artifacts repositories create --location="${LOCATION}" --repository-format=docker --project="${PROJECT_ID}" "${REPOSITORY_NAME}"
```

### Creating and pushing a new package

Creating a new package is no different. But when ready to publish, the `kpt pkg push` command 
is used instead of source control operations.

```shell
# A package in a new directory
mkdir hello-world
kpt pkg init hello-world --description="A simple blueprint"

# Store the contents in the repository, tagged as v1
kpt pkg push hello-world --image=${REPOSITORY}/hello-world:v1

# The local files are not needed any more, pushing has stored them all
rm -r hello-world
```

### Pulling and updating a package

Because the package folder was discarded earlier, the `kpt pkg pull` command 
is used to place the contents of a particular version at a location. These set of
commands may be run in `Cloud Build` steps as well, if you are automating the
publication of packages as part of a CI/CD process.

```shell
# Recreate the folder and extract the pulled image
kpt pkg pull hello-world --image=${REPOSITORY}/hello-world:v1

# Add a sub-package from a git repo
kpt pkg get https://github.com/GoogleCloudPlatform/blueprints.git/catalog/bucket hello-world/my-bucket

# Render to be sure contents are hydrated, and push to a new version tag
kpt pkg render hello-world
kpt pkg push hello-world --image=${REPOSITORY}/hello-world:v2
```

Similar to container image tags, the package image tags like `:v1` and `:v2` above may be used any
number of ways based on your preferred workflows. The tag `:latest` is used by default if all
pulls and pushes should read from and overwrite the same location. Semantic tags like `:draft` and
environmental tags like `:dev`, `:qa`, and `:prod` may be also used.

No matter what tags are used, the image repository and `kpt` cli will treat them as an alphanumeric 
label.

### Using OCI repository as an upstream

In addition to providing storage for packages, an OCI registry may also 
be used as a source of upstream images to clone. The `oci://` prefix on this
command is required to ensure 

```shell
# Clone the hello-world v1 blueprint into a new folder
kpt pkg get oci://${REPOSITORY}/hello-world:v1 greetings-planet

# Push the results to the repository, using default `latest` tag in this example
kpt pkg push greetings-planet --image=${REPOSITORY}/greetings-planet
```

Looking in the `greetings-planet/Kptfile` at this point will show that
the `hello-world:v1` image is the `upstream`, and the `upstreamLock` will show
exactly the digest that this clone is up-to-date with. 

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: greetings-planet
upstream:
  type: oci
  oci:
    image: us-docker.pkg.dev/kpt-demo-73823/blueprints/hello-world:v1
  updateStrategy: resource-merge
upstreamLock:
  type: oci
  oci:
    image: us-docker.pkg.dev/kpt-demo-73823/blueprints/hello-world@sha256:1632e00af3fe858c5e3b3f9e75c16e6327449155
```

### Adding a subpackage from an OCI upstream subfolder

Often a folder inside a package is meant to be used as a way to create "more of the same".

To use an OCI image subfolder as the source of a subpackage, the path is added in a 
way that's distinct from the image itself.

```shell
# Clone the hello-world v1 blueprint into a new folder
kpt pkg get oci://${REPOSITORY}/hello-world//my-bucket:v1 greetings-planet/another-bucket
```

The `greetings-planet` package will now contain both a `greetings-planet/my-bucket` as well as a
`greetings-planet/another-bucket` folder. The contents in locations will now both receive changes
when the upstream `hello-world/my-bucket` is updated.

### Updating package with upstream changes

The value of the upstream image tag is used to `kpt pkg update` to a specific version.
This works no matter if the tag appears to look like a version number or not.

```shell
# Update the greetings-planet by applying any differences between the upstreamLock digest and the `v2` tag
kpt pkg update greetings-planet@v2

# Overwrite the `greetings-planet:latest` image with the folder contents
kpt pkg push greetings-planet --image=${REPOSITORY}/greetings-planet
```

The Kptfile will now show that the `upstream` and `upstreamLock` have both been changed.

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: greetings-planet
upstream:
  type: oci
  oci:
    image: us-docker.pkg.dev/kpt-demo-73823/blueprints/hello-world:v2
  updateStrategy: resource-merge
upstreamLock:
  type: oci
  oci:
    image: us-docker.pkg.dev/kpt-demo-73823/blueprints/hello-world@sha256:a6f1ed69c6ab51e2a148f6d4926bccb24c843887
```

Just like with a git upstream, it is also possible to `kpt pkg update` without providing a different
tag or version value. This is similar to pulling from a remote branch where the name of the branch does 
change but the latest commit on that branch is a different hash.

In that case the Kptfile `upstream` tag will not change, but if that tag has been overwritten with
new contents then the differences between `upstreamLock` and current contents will be applied to the local copy,
and the `upstreamLock` will be changed tot he current digest.

### Updating package to a specific upstream push

Much like you can `kpt pkg update` to a specific commit hash in git, you can update to an
exact image digest with OCI. This is an even more precise reference than by a version number because,
like git commits, the image digest is based on the package file contents and cannot be forged or altered.

```shell
# Update instread to an exact image digest of the upstream location
kpt pkg update greetings-planet@sha256:3b42daa41102fa83bce07bd82a72edcd691868d6
```

The resulting Kptfile in the local folder will look like this.

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: greetings-planet
upstream:
  type: oci
  oci:
    image: us-docker.pkg.dev/kpt-demo-73823/blueprints/hello-world@sha256:3b42daa41102fa83bce07bd82a72edcd691868d6
  updateStrategy: resource-merge
upstreamLock:
  type: oci
  oci:
    image: us-docker.pkg.dev/kpt-demo-73823/blueprints/hello-world@sha256:3b42daa41102fa83bce07bd82a72edcd691868d6
```

## Open Issues/Questions

> Please list any open questions here in the following format:
> 
> ### \<Question\>
> 
> Resolution: Please list the resolution if resolved during the design process or
> specify __Not Yet Resolved__

### What additional container registries should be supported?

The protocol and information is the same. It would mainly be a question
of how the credentials for the call are provided.

### What commands on exising kpt binary will work on Kptfile with `oci`

It may be possible `kpt` commands that to not process `upstream` structures
may not require update to work correctly. `kpt fn` and `kpt live` commands
should be tested to see how they behave.

## Alternatives Considered

If there is an industry precedent or alternative approaches please list them 
here as well as citing *why* you decided not to pursue those paths.

### \<Approach\>

Links and description of the approach, the pros and cons identified during the 
design. 
