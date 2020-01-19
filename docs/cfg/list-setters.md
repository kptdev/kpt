## set

List setters for Resources.

![alt text][demo]

[demo-script](../../gifs/cfg-set.sh)

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

[demo]: https://storage.googleapis.com/kpt-dev/docs/cfg-set.gif "kpt cfg set"