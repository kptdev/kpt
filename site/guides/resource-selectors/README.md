## Resource Selectors

In this guide we will see how to target specific resources for running functions 
both declaratively and imperatively.

Fetch the `wordpress` package:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.7
```

For declarative workflows, selectors follow OR of AND(s) approach where, within 
each selector, the filters are ANDed and the selected resources are UNIONed with 
other selected resources. Please go through [Kptfile schema] for list of all selector 
properties.

Selectively add annotations only to the `mysql` subpackage resources by 
declaring `set-annotations` function in `wordpress/Kptfile` and `selectors` option:

```yaml
# wordpress/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: set-labels:v0.1
      configMap:
        app: wordpress
    - image: set-annotations:v0.1
      configMap:
        tier: mysql
      selectors:
        - packagePath: ./mysql
```

Render the resources:

```shell
kpt fn render wordpress
```

Selectively add name-prefix to only `Deployment` resources with specific `name` OR
`Deployment` resources with specific `name` by declaring `ensure-name-substring` 
function in `wordpress/Kptfile` with combination of `selectors`:

```yaml
# wordpress/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: set-labels:v0.1
      configMap:
        app: wordpress
    - image: ensure-name-substring:v0.1
      configMap:
        prepend: dev-
      selectors:
        - kind: Deployment
          name: wordpress
        - kind: Service
          name: wordpress
```

Render the resources:

```shell
kpt fn render wordpress
```

For imperative workflows, the filters are ANDed to select resources. Imperative 
equivalent of setting the name-prefix to only `Deployment` resources with specific `name`:

```shell
kpt fn eval wordpress/ -i ensure-name-substring:v0.1 --kind Deployment --name wordpress -- prepend=dev-
```

Please go through [eval flags] for list of all selector properties.

[Kptfile schema]: https://kpt.dev/reference/schema/kptfile/
[eval flags]: https://kpt.dev/reference/cli/fn/eval/?id=flags
