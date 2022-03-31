# Title

* Author(s): Mike Borozdin, @mikebz
* Approver: Mengqi, Sunil

## Why

More people are getting Apple M1 machines.  The current docker images for
functions do not work.  Here is an example:
https://github.com/GoogleContainerTools/kpt/issues/2874
While it's not a problem for CI/CD pipelines where architecture is mostly amd64
for client development purposes the users are stuck.

## Design

Building arm64 and amd64 multi platform images is possible:
https://www.docker.com/blog/multi-arch-build-and-images-the-simple-way/
but it has it's caveats.  The build is done in a builder which needs to be 
created:
https://docs.docker.com/engine/reference/commandline/buildx_create/
The caveat is that the multi platform images do not load into docker images:
https://github.com/docker/buildx/blob/master/docs/reference/buildx_build.md#-load-the-single-platform-build-result-to-docker-images---load

Right now the build pipeline has several steps:
1) build
2) tag
3) push

In different scenarios only the first two steps are executed.  If we switch to 
`buildx` we will need to build/tag/push in one step.  Locally developers 
might want to build and tag images locally.

However if we make the changes needed to the shell scripts that produce the
images all that the users of docker images will need to do is just update
to the next version of the function image.  Docker promises to use the right 
architecture automatically.

## User Guide

The user guide for using the new images should not change.  People using
amd64 systems (Linux or Mac) can continue using the same systems.  People using
arm64 systems can already get the arm64 kpt binary and will update their
pipelines to the next version of functions.  The functions should select the 
right architecture to run.

## Open Issues/Questions

Please list any open questions here in the following format:

## Alternatives Considered

An alternative to using a multi architecture images is to build a special image
for each platform and then tag them differently.  The benefits of this are:
- users can select the right architecture in their Kptfile or imperability 
invoking the right function image.

The problem is that most of the time CI/CD systems are amd64 linux and the arm64
is primarily for the convenience of client systems.  The users most likely
do not want to change their hydration pipeline definition from client to CI/CD
systems.

NOTE: we have done a test on how the images look in the container registry.
buildx builds two images and the individual images are not any bigger.

### \<Approach\>

Links and description of the approach, the pros and cons identified during the 
design.
