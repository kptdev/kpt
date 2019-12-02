# kpt docs

## Installation
    
    # install kpt
    go get github.com/GoogleContainerTools/kpt/internal/cmd

    # install kustomize
    go get sigs.k8s.io/kustomize/kustomize
    
## Commands

- [init](commands/init.md): Initialize suggested package meta for a local config directory
- [desc](commands/desc.md): Display package descriptions
- [get](commands/get.md): Fetch a package from a git repository
- [man](commands/man.md): Format and display package documentation if it exists
- [sync](commands/sync.md): Fetch dependencies using a manifest rather than commands
- [update](commands/update.md): Update a local package with changes from a remote source repo

## Tutorials

1. [get](tutorials/fetch-a-package.md): Fetch a package from a remote git repository
1. [view](tutorials/working-with-local-packages.md): View fetched package information
1. [update](tutorials/update-a-local-package.md): Update a previously fetched package 
1. [publish](tutorials/publish-a-package.md): Publish a new package

