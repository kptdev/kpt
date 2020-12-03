# kpt roadmap for 2021

Last updated: December 2020

## Packaging
We want to provide better support for nested packages, in particular making
sure that _kpt pkg get_ and _kpt pkg update_ properly respect package boundaries.
As part of this, we are also taking another look at the _kpt pkg sync_
functionality with the goal of incorporating this into the regular packaging
commands. 

Kpt packages currently can contain directories and the only way to
determine the boundaries for subpackages are by looking for the presence of
Kptfiles. This has proven challenging at times, in particular around merging
changes when the boundaries of packages change. We are exploring making every
directory a separate package, regardless of whether there is a Kptfile.

## Configuration and setters
Setters and substitutions are useful in many situations, but we are looking to
make them simpler to use and more flexible. We are looking at options for
making setters more declarative, in particular making it possible to update
the setter values directly in a file and then propagate all values into the
resources in the package. We have also seen that substitutions don't always
work well for large numbers of resources, so we are looking to make them simpler
and more flexible. We are also introducing a Search and Replace functionality
that will make bulk edits across manifests much easier.

## Functions
Kpt functions in their current format are useful for performing individual
tasks, but they are hard to leverage for more complex use-cases such as
hydrating DRY configuration or generating variants of a package for different
environments. We are looking to introduce the concept of a pipeline of kpt
functions as an optional part of a kpt package. A pipeline declares operations
that will be performed on the resources declared within the package as well on
other packages using recursive resolution. It will also provide flexibility to
output the results of the operations using different sinks modes: in-place,
stdout, or an external directory. 

## Live
_kpt live_ is currently using a ConfigMap to store inventory information. This is
convenient as it doesn’t require any CRDs, but it has challenges around encoding
of the GroupKind, name and namespace, and it doesn’t allow us to easily add
additional metadata about a package, such as the package version. We are working
on migrating kpt to use a CRD for storing the inventory information.

We have also started looking at how to best handle nested packages during apply.
Currently the nested structure is not reflected in the live state on the
apiserver. 

The majority of the code for kpt live is in the cli-utils repo. Our
goal is to keep this code independent of kpt and provide the apply logic as a
library that can be used by other tools. The API is still going through some
changes, but we are actively working to stabilize it.
