## set

List setters for Resources.

### Synopsis

    kpt config list-setters DIR [NAME]

  DIR

    A directory containing Resource configuration.

  NAME

    Optional.  The name of the setter to display.

### Examples

  Show setters:

    $ kpt config list-setters DIR/
        NAME      DESCRIPTION   VALUE     TYPE     COUNT   SETBY  
    name-prefix   ''            PREFIX    string   2
