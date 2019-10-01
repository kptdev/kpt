## tutorials 5-publish-a-package

Publish a new package

### Synopsis

While packages may be published as directories of raw Configuration,
kpt supports blessing a directory with additional package metadata that can benift
package discovery.

	kpt bless my-package/ --name my-package --description 'fun new package'
	git add my-package && git commit -m 'new kpt package'
	git push origin master

  This blessed the package by creating a Kptfile and MAN.md.  The MAN.md may be
  modified to include package documentation, which can be displayed with 'kpt man local-copy/'


```
tutorials 5-publish-a-package [flags]
```

### Options

```
  -h, --help   help for 5-publish-a-package
```

### SEE ALSO

* [tutorials](tutorials.md)	 - Contains tutorials for using kpt

