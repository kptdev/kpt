## Publish a Package

This tutorial covers how to publish a package to a remote source.

### Synopsis

Any bundle of Resource configuration may be published as a package using `git`.

`kpt init` initializes a directory with optional package metadata such as a
package documentation file.

### Examples

    kpt init my-package/ --name my-package --description 'fun new package'
    git add my-package && git commit -m 'new kpt package'
    git push origin master
