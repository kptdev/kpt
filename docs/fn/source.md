## source

Explicitly specify an input source

### Synopsis

Implements a Source by reading configuration and writing to command stdout.

    kpt fn source [DIR...]

  DIR:
    One or more paths to local directories.  Contents from directories will be concatenated.
    If no directories are provided, source will read from stdin as if it were a single file.

`source` emits configuration to act as input to a function

### Examples

    # print to stdout configuration formatted as an input source
    kpt fn source DIR/

    # run a function using explicit sources and sinks
    kpt fn source DIR/ | kpt run --image gcr.io/example.com/my-fn | kpt fn sink DIR/
