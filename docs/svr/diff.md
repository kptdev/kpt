## diff

Block until the Resources are current, printing status changes as events

### Synopsis

Diff local configuration against the applied cluster state.

 Output is always YAML.

Env Vars:

  KUBECTL_EXTERNAL_DIFF:
    Environment variable can be used to select your own diff command.
    By default, the "diff" command available in your path will be run with
    "-u" (unified diff) and "-N" (treat absent files as empty) options.

### Examples

  # Diff resources included in pod.json.
  kpt svr diff -R -f DIR/

  # Diff file read from stdin
  cat service.yaml | kpt svr diff -f -
