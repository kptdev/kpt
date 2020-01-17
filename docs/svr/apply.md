## apply

Apply local Resource configuration to a cluster.

### Synopsis

Set the desired state of a cluster to match the locally defined Resource configuration.

### Examples

  # Apply all resources under a directory
  kpt svr apply -R -f DIR/

  # Apply resources from stdin
  cat service.yaml | kpt svr apply -f -
