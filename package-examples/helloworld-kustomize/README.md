helloworld-kustomize
==================================================

# NAME

  helloworld-kustomize

# SYNOPSIS

  kubectl apply --recursive -f helloworld-kustomize

# Description

a sample where the upstream package contains a kustomize patch that contains a setter.  One can run the pipeline in the package
```
kpt fn render .
```
and observe the message in the package change.  You can also see documentation on `apply-setters` function to see a full range
of ways to apply the changes.

# SEE ALSO

