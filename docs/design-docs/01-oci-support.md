# Title

* Author(s): \<your name\>, \<your github alias\>
* Approver: \<kpt-maintainer\>

>    Every feature will need design sign off an PR approval from a core
>    maintainer.  If you have not got in touch with anyone yet, you can leave
>    this blank and we will try to line someone up for you.

## Why

Please provide a reason to have this feature.  For best results a feature should
be addressing a problem that is described in a github issue.  Link the issues
in this section.  The more user requests are linked here the more likely this
design is going to get prioritized on the roadmap.

It's good to include some background about the problem, but do not use that as a
substitute for real user feedback.



https://github.com/GoogleContainerTools/kpt/issues/2300


## Design

Please describe your solution. Please list any:

* new config changes
* interface changes
* design assumptions

For a new config change, please mention:

* Is it backwards compatible? If not, what is the migration and deprecation 
  plan?

### Config chages

The `Kptfile` properties `upstream` and `upstreamLock` have `oci` properties added alongside the `git` properties. The `type` property also has the string `oci` added as an accepted value.

Old `Kptfile` will `git` upstream be compatible with new verions of `kpt`, the file structure and git functionality is unchanged.

New `Kptfile` with `git` upstream will be compatible with old versions of `kpt`.

New `Kptfile` with `oci` upstream will not be compatible with old versions of `kpt`. The `kpt` command being used will need to be updated.  

### Command changes

* `kpt pkg get`

The argument that determines upstream today is parsed into `repo`, `ref`, and `path`, and is implicitly a `git` location.

To support `oci`, it will be necessary to extract different values in a way that's unambiguous. Unfortunately, OCI image names have no Uri prefix, and are indistinguishable from a valid path or file name.

To solve this, using [Helm](https://helm.sh/docs/topics/registries/#other-subcommands) as an example, the prefix `oci://` can be used. This ensures that selecting `oci` protocol isn't accidental, and it won't collide with other location formats that may be added.

```
$ kpt pkg get oci://us-docker.pkg.dev/my-project-id/my-repo/flowers:v3
```

Because OCI image reference already has a convention for `image:tag` references, using `:v3` should be used instead of `@v3` for version. It will be more intuitive how it relates to the registry, and easier to cut and paste values.

Sub-Packages



* `kpt pkg update`
* `kpt pkg diff`


## User Guide

This section should be written in the form of a detailed user guide describing 
the user journey. It should start from a reasonable initial state, often from 
scratch (Instead of starting midway through a convoluted scenario) in order 
to provide enough context for the reader and demonstrate possible workflows. 
This is a form of DDD (Documentation-Driven-Development), which is an effective 
technique to empathize with the user early in the process (As opposed to 
late-stage user-empathy sessions).

This section should be as detailed as possible. For example if proposing a CLI 
change, provide the exact commands the user needs to run, along with flag 
descriptions, and success and failure messages (Failure messages are an 
important part of a good UX). This level of detail serves two functions:

It forces the author and the readers to explicitly think about possible friction
points, pitfalls, failure scenarios, and corner cases (“A measure of a good 
engineer is not how clever they are, but whether they think about all the 
corner cases”). Makes it easier to author the user-facing docs as part of the 
development process (Ideally as part of the same PR) as opposed to it being an 
afterthought.

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

Please list any open questions here in the following format:

### \<Question\>

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__

## Alternatives Considered

If there is an industry precedent or alternative approaches please list them 
here as well as citing *why* you decided not to pursue those paths.

### \<Approach\>

Links and description of the approach, the pros and cons identified during the 
design. 