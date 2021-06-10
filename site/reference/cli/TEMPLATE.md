---
title: "`cmd`"
linkTitle: "cmd"
type: docs
description: >
One line description of the command
---
<!--mdtogo:Short
    One line description of the command
-->

Short description of the command. The command should be referred to verbatim and
not be capitalized, for example `get`, `update`, `fn`.
The documentation for each command should be focused and concise. Any fundamental
concepts or concerns cutting across many command should be covered in the kpt book
and the reference docs should provide deep links to the relevant content.
Args, flags, and env vars should be listed in alphabetical order.

### Synopsis
<!--mdtogo:Long-->
```
kpt <cmd group> <cmd> <ARGS> [flags]
```

#### Args
```
EXAMPLE_ARG:
  Description of the arg. This should include information about whether the ARG
  is required and if it is not, what would be the default value. It should use
  correct grammar, including capitalization and punctuation.
```

#### Flags
```
--example-flag:
  Description of the flag, including the default value. It should use correct 
  grammar, including capitalization and punctuation.
```

#### Env Vars
```
EXAMPLE_ENV_VAR:
  Description of the env variable. It should use correct 
  grammar, including capitalization and punctuation.
```
<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->
```shell
# Examples of how to use the command. This should include the most common use-cases
# and include an example where the local package argument is not provided.
# The description of each example should use correct grammar, including capitalization
# and punctuation.
kpt <cmd group> <cmd> ...
```
<!--mdtogo-->