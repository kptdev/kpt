---
title: "API Conventions"
linkTitle: "API Conventions"
weight: 2
type: docs
description: >
   Kpt API Conventions
---

## CLI Conventions

Following are the CLI conventions used for building kpt.  This is a living document that should
be iterated upon.

### Command IO Conventions

- Commands that read resources should be able to read them from files, directories or stdin
  - It should be possible to pipe `kubectl get -o yaml` to the input of these commands
- Commands that write resources should be able to write them to files, directories or stdout
  - It should be possible to pipe the output of these commands to `kubectl apply -f -`
- It should be possible to compose commands by piping them together
  - Metadata (such as which file a resource was read from) may need to be persisted to the
    resources (e.g. as annotations) or as part of the output format (e.g. ResourceList) so
    this state isn't lost between piped commands.
  - e.g. it should be possible to read resources from one command, and pipe them to another command
    which will write them back to files.

### Arguments, subcommands and flags

- Directories and files should be arguments
  - e.g. `kpt live apply DIR/` vs `kubectl apply -f DIR/`
  - This makes it easier to compose with tools such as `find` and `xargs`
- Subcommands commands should be used in favor of mutually exclusive or complex flags
  - e.g. `kubectl create cronjob --schedule=X` vs `kubectl run --generator=cronjob --schedule=X`
  - This simplifies the documentation (subcommands support more documentation options than flags do)
- Features which are alpha should be guarded behind flags and documented as alpha in the command
  help or flag help.

### Documentation

Documentation should be compiled into the command help itself.

- Reference documentation should be built into the help for each command
- Guides and concept docs should be built into their own "help" commands
- Reference documentation should have asciinema-style "images" that demonstrate
  how the commands are used.

## Resource Annotations

kpt uses the following annotations to store resource metadata.

### Resource IO Annotations

#### config.kubernetes.io/path

`config.kubernetes.io/path` stores the file path that the resource was read from.

- When reading resources, if reading from a directory kpt should annotate each resource with the path of the file it was read from
- When writing resources, if writing to a directory kpt should read the annotation and write each to the file matching the path

#### config.kubernetes.io/index

`config.kubernetes.io/index` stores the index into the file that the resource was read from.

- When reading resources, if reading from a file kpt should annotate the resource with the index into the file
- When writing resources, if writing to a file kpt should write the resources in order specified by the indexes

### Functions Annotations

#### config.kubernetes.io/function

`config.kubernetes.io/function` indicates that the resource may be provided to the specified function
as the ResourceList.functionConfig.

## Next Steps

- See the [Configuration IO API Semantics] for when to use resource annotations.
- Learn more about [functions concepts].
- Consult the [FAQ] for answers to common questions.

[Configuration IO API Semantics]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/config-io.md
[functions concepts]: ../functions/
[FAQ]: ../../faq/
