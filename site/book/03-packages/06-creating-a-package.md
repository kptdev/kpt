Creating a new package is simple: create a new directory and [author resources]:

```shell
$ mkdir awesomeapp
# Create resources in awesomeapp/
```

For convenience, you can use `pkg init` command to create a minimal `Kptfile`
and `README` files:

```shell
$ kpt pkg init awesomeapp
writing Kptfile
writing README.md
```

?> Refer to the [init command reference][init-doc] for usage.

The `info` section of the `Kptfile` contains some optional package metadata you
may want to set. These fields are not consumed by any functionality in kpt:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: awesomeapp
info:
  description: Awesomeapp solves all the world's problems in half the time.
  site: awesomeapp.example.com
  emails:
    - jack@example.com
    - jill@example.com
  license: Apache-2.0
  keywords:
    - awesome-tech
    - world-saver
```

[author resources]: /book/03-packages/03-editing-a-package
[init-doc]: /reference/cli/pkg/init/
