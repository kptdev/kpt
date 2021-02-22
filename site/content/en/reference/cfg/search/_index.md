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
Match by path expression of a field. Path expressions are used to deeply navigate
and match particular yaml nodes. Please note that the path expressions are not
regular expressions.

--put-value
Set or update the value of the matching fields. Input can be a pattern for which
the numbered capture groups are resolved using --by-value-regex input.

--put-comment
Set or update the line comment for matching fields. Input can be a pattern for
which the numbered capture groups are resolved using --by-value-regex input.

--recurse-subpackages (default: true)
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
$ kpt cfg search DIR/ --by-value-regex 'ngnix-.*'
```

```sh
# Matches field with path "spec.namespaces" set to "bookstore":
$ kpt cfg search DIR/ --by-path 'metadata.namespace' --by-value 'bookstore'
```

```sh
# Matches fields with name "containerPort" arbitrarily deep in "spec" that have value of 80:
$ kpt cfg search DIR/ --by-path 'spec.**.containerPort' --by-value 80
```

```sh
# Set namespaces for all resources to "bookstore", even namespace is not set on a resource:
$ kpt cfg search DIR/ --by-path 'metadata.namespace' --put-value 'bookstore'
```

```
# Search and Set multiple values using regex numbered capture groups
$ kpt cfg search DIR/ --by-value-regex 'something-(.*)' --put-value 'my-project-id-${1}'
metadata:
  name: something-foo
  namespace: something-bar
...
metadata:
  name: my-project-id-foo
  namespace: my-project-id-bar
```

```sh
# Put the setter pattern as a line comment for matching fields.
$ kpt cfg search DIR/ --by-value 'my-project-id-foo' --put-comment 'kpt-set: ${project-id}-foo'
metadata:
  name: my-project-id-foo # kpt-set: ${project-id}-foo

# Setter pattern comments can be added to multiple values matching a regex numbered capture groups
$ kpt cfg search DIR/ --by-value-regex 'my-project-id-(.*)' --put-comment 'kpt-set: ${project-id}-${1}'
metadata:
  name: my-project-id-foo # kpt-set: ${project-id}-foo
  namespace: my-project-id-bar # kpt-set: ${project-id}-bar
```

Supported Path expressions:

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

<!--mdtogo-->

### Output format

```sh
$ kpt cfg search my-dir/ --by-value 3
my-dir/path/to/file1.yaml
fieldPath: spec.replicas
value: 3

my-dir/path/to/file2.yaml
fieldPath: spec.foo
value: 3

Matched 2 field(s)
```

[this document]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md#declarative-configuration
