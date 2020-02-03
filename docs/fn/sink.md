## sink

Explicitly specify an output sink

### Synopsis

Implements a Sink by reading command stdin and writing to a local directory.

    kpt fn sink DIR

  DIR:
    Path to local directory.

`sink` writes its input to a directory

### Examples

    # run a function using explicit sources and sinks
    kpt fn source DIR/ | kpt run --image gcr.io/example.com/my-fn | kpt fn sink DIR/
