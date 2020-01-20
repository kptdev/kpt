## set

List setters for Resources.

![alt text][tutorial]

    kpt tutorial cfg set

[tutorial-script]

### Synopsis

    kpt cfg list-setters DIR [NAME]

  DIR

    A directory containing Resource configuration.

  NAME

    Optional.  The name of the setter to display.

### Examples

  Show setters:

    $ kpt cfg list-setters DIR/
        NAME      DESCRIPTION   VALUE     TYPE     COUNT   SETBY
    name-prefix   ''            PREFIX    string   2

###

[tutorial]: https://storage.googleapis.com/kpt-dev/docs/cfg-set.gif "kpt cfg set"
[tutorial-script]: ../../gifs/cfg-set.sh
