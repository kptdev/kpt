## sink

Implement a Sink by writing input to a local directory.

### Synopsis

Implement a Sink by writing input to a local directory.

    kpt fn sink DIR

  DIR:
    Path to local directory.

`sink` writes its input to a directory

### Examples

    kpt fn source DIR/ | your-function | kpt fn sink DIR/
