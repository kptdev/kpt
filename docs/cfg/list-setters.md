## set

List setters for Resources.

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg-set.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

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

[tutorial-script]: ../gifs/cfg-set.sh
