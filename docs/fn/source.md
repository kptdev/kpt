## source

Implement a Source by reading a local directory.

### Synopsis

Implement a Source by reading a local directory.

    kpt fn source DIR

  DIR:
    Path to local directory.

`source` emits configuration to act as input to a function

### Examples

    # emity configuration directory as input source to a function
    kpt fn source DIR/

    kpt fn source DIR/ | your-function | kpt fn sink DIR/
