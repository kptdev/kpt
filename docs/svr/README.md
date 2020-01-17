## svr

Send Resource request to a Kubernetes apiserver

### Synopsis

In order to make use of Resource configuration, it must be sent to an apiserver where it
will be actuated by Kubernetes Controllers.

### Commands

**[apply](apply.md)**:
- Declaratively create, update and delete Kubernetes objects from a Resource configuration directory

**[diff](diff.md)**:
- Display differences between local Resource configuration, and live Kubernetes Resources

**[status fetch](fetch.md)**:
- Display current Resource status as a table

**[status wait](wait.md)**:
- Display current Resource status as a table, and block until live state matches the applied state.

**[status events](fetch.md)**:
- Print Resource status events until live state matches the applied state.

### Examples

    kpt svr apply -R -f DIR/

    kpt svr diff -R -f DIR/

    kpt svr wait DIR/