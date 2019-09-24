## kpt duck-typed set-env

Set an environment variable on a container

### Synopsis

Set an environment variable on a container.

Args:

  NAME:
    Name of the Resource and Container on which to set the environment variable.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed set-env NAME [flags]
```

### Examples

```
kpt  set env NAME --name ENV_NAME --value ENV_VALUE
```

### Options

```
  -h, --help           help for set-env
      --name string    the environment variable name
      --value string   the environment variable value
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

