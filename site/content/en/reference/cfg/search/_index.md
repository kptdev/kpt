---
title: "Search"
linkTitle: "search"
weight: 4
type: docs
description: >
  Search and optionally replace fields across all resources.
---

<!--mdtogo:Short
    Search and optionally replace fields across all resources.
-->

{{% pageinfo color="primary" %}}

This feature is under development. Please set environment variable
`KPT_ENABLE_SEARCH_CMD` to `true` on your local to enable this feature.

{{% /pageinfo %}}

There is a spectrum of configuration customization techniques as described in
[this document]. One of the most basic and simplest to understand is
Search and Replace: The user fetches a package of configuration, searches all
the files for fields matching a criteria, and replaces their values.

Search matchers are provided by flags with `--by-` prefix. When multiple matchers
are provided they are AND’ed together. `--put-` flags are mutually exclusive.

### Synopsis

<!--mdtogo:Long-->

```
kpt cfg search DIR [flags]
```

#### Args

```
DIR:
  Path to a package directory
```

<!--mdtogo-->

#### Flags

```sh
--by-value
Match by value of a field.

--by-value-regex
Match by Regex for the value of a field. The syntax of the regular expressions
accepted is the same general syntax used by Go, Perl, Python, and other languages.
More precisely, it is the syntax accepted by RE2 and described at
https://golang.org/s/re2syntax. With the exception that it matches the entire
value of the field by default without requiring start (^) and end ($) characters.

--by-path
Match by path expression of a field. See Path Expressions.

--put-literal
Set or update the value of the matching fields with the given literal value.

--put-pattern
Put the setter pattern as a line comment for matching fields.

--recurse-subpackages
Search recursively in all the nested subpackages.
```

### Examples

<!--mdtogo:Examples-->

```sh
# Matches fields with value "3":
$ kpt cfg search DIR/ --by-value 3
```

```sh
# Matches fields with value prefixed by "nginx-":
$ kpt cfg search DIR/ --by-value-regex ngnix-.*
```

```sh
# Matches field with path "spec.namespaces" set to "bookstore":
$ kpt cfg search DIR/ --by-path metadata.namespace --by-value bookstore
```

```sh
# Matches fields with name "containerPort" arbitrarily deep in "spec" that have value of 80:
$ kpt cfg search DIR/ --by-path spec.**.containerPort --by-value 80
```

```sh
# Set namespaces for all resources to "bookstore", even namespace is not set on a resource:
$ kpt cfg search DIR/ --by-path metadata.namespace --put-literal bookstore
```

```sh
# Create a setter for a GCP project ID and parameterize all field values with the
# prefix "my-project-id" with that setter:
$ kpt cfg create-setter DIR/ project-id my-project-id

Assume there are two fields with value "my-project-id-foo" and "my-project-id-bar"

$ kpt cfg search DIR/ --by-value-regex my-project-id-* --put-pattern \${project-id}-*
This will add the setter comment to matching fields:
my-project-id-foo # ${project-id}-foo
...
my-project-id-bar # ${project-id}-bar
```

<!--mdtogo-->

### Output format

```sh
$ kpt cfg search my-dir/ --by-value 80
my-dir/
matched 2 field(s)
my-dir/file1.yaml:  spec.ports[0].port: 80
my-dir/file2.yaml:  spec.template.spec.containers[0].ports[0].containerPort: 80
```

### Path Expressions

The following Path expressions are supported:

```sh
a.b.c

a:
  b:
    c: thing # MATCHES
```

```sh
a.*.c

a:
  b1:
    c: thing # MATCHES
    d: whatever
  b2:
    c: thing # MATCHES
    f: something irrelevant
```

```sh
a.**.c

a:
  b1:
    c: thing1 # MATCHES
    d: cat
  b2:
    c: thing2 # MATCHES
    d: dog
  b3:
    d:
    - f:
        c: thing3 # MATCHES
        d: beep
    - f:
        g:
          c: thing4 # MATCHES
          d: boop
    - d: mooo
```

```sh
a.b[1].c

a:
  b:
  - c: thing0
  - c: thing1 # MATCHES
  - c: thing2
```

```sh
a.b[*].c

a:
  b:
  - c: thing0 # MATCHES
    d: what..ever
  - c: thing1 # MATCHES
    d: blarh
  - c: thing2 # MATCHES
    f: thingamabob
```

[this document]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md#declarative-configuration
