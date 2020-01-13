## sink

Implement a Sink by writing input to a local directory.

### Synopsis

Implement a Sink by writing input to a local directory.

    kpt config sink DIR

  DIR:
    Path to local directory.

`sink` writes its input to a directory

### Examples

    kpt config source DIR/ | your-function | kpt config sink DIR/
