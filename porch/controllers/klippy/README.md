# Overview

klippy is a controller that proposes creating packages as a child of a
parent package, where all the bindings can be satisfied.

The idea is that packages in this "proposed by robot" state can
provide a guided authoring experience, and we can communicate these
paths through the existing API.

The controller and the idea of package proposals should be considered
exploration / experimental.

# Bindings

The idea of bindings is that a blueprint author can include some
objects that are specially marked with the
`config.kubernetes.io/local-config` = `binding` annotation.  Those are
binding objects.

Because those binding objects are `local-config`, they will not be
applied to the cluster as part of the package.  Instead, those objects
normally come from a parent package.

The value of a binding object comes when we replace the placeholder
values with the actual values from the parent.  We do a semantically
aware rename, so - for example - if a binding objects is a namespace,
all the objects in the binding placeholder namespace would be changed
to be in the newly bound namespace.

If the object is something like a ConfigMap, we would update all the
references to that ConfigMap, for example in pod volumes.

# klippy: auto-binding

The idea of the klippy controller therefore is to eagerly look for
places where we can instantiate a child package under a parent
package, where all the bindings can be satisfied.

We match bindings based on the Group/Version/Kind.  Additionally, if
the binding object has labels, we'll look for those labels on the
parent package object (this was needed because otherwise we were
over-proposing on common objects like Namespaces, in practice we need
some sort of "intent" label on Namespaces or GCP Projects/Folders,
that indicates what we expect them to contain)
