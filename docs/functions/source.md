## source

Implement a Source by reading a local directory.

### Synopsis

Implement a Source by reading a local directory.

    kpt config source DIR

  DIR:
    Path to local directory.

`source` emits configuration to act as input to a function

### Examples

    # emity configuration directory as input source to a function
    kpt config source DIR/

    kpt config source DIR/ | your-function | kpt config sink DIR/
