## sink

Implement a Sink by writing input to a local directory.

### Synopsis

Implement a Sink by writing input to a local directory.

    kpt functions sink DIR

  DIR:
    Path to local directory.

`sink` writes its input to a directory

### Examples

    kpt functions source DIR/ | your-function | kpt functions sink DIR/
