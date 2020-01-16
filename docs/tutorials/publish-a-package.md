## Publish a Package

How to publish a package to a remote source

### Synopsis

Any bundle of Resource configuration may be published as a package using `git`.

`kpt pkg init` initializes a directory with optional package metadata such as a
package documentation file.

### Examples

    kpt pkg init my-package/ --name my-package --description 'fun new package'
    git add my-package && git commit -m 'new kpt package'
    git push origin master
