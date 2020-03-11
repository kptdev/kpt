## set

List configured field setters

### Synopsis

    kpt cfg list-setters DIR [NAME]

  DIR

    A directory containing a Kptfile.

  NAME

    Optional.  The name of the setter to display.

### Examples

    # list the setters in the hello-world package
    kpt cfg list-setters hello-world/
      NAME     VALUE    SET BY    DESCRIPTION   COUNT  
    replicas   4       isabella   good value    1   

###

[tutorial-script]: ../gifs/cfg-set.sh
