## init

initialize a package by creating a local file

### Synopsis

    kpt live init DIRECTORY [flags]

The init command initializes the package by locally creating a template
file. When applied, this template file is used to store the state of all
applied objects in a package. This file is necessary for other live 
commands (apply/preview/destroy) to work correctly.

Args:
  DIRECTORY:
    One directory that contain k8s manifests. The directory
    must contain exactly one ConfigMap with the grouping object annotation.

Flags:
  group-name:
    String name to group applied resources. Must be composed of valid
    label value characters. If not specified, the default group name
    is generated from the package directory name.
