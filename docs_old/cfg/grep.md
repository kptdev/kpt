## grep

Find resources by field value

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg-grep.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    kpt tutorial cfg grep

[tutorial-script]

### Synopsis

    kpt cfg grep QUERY DIR

  QUERY:
    Query to match expressed as 'path.to.field=value'.
    Maps and fields are matched as '.field-name' or '.map-key'
    List elements are matched as '[list-elem-field=field-value]'
    The value to match is expressed as '=value'
    '.' as part of a key or value can be escaped as '\.'

  DIR:
    Path to local directory.

### Examples

    # find Deployment Resources
    kpt cfg grep "kind=Deployment" my-dir/

    # find Resources named nginx
    kpt cfg grep "metadata.name=nginx" my-dir/

    # use tree to display matching Resources
    kpt cfg grep "metadata.name=nginx" my-dir/ | kpt cfg tree

    # look for Resources matching a specific container image
    kpt cfg grep "spec.template.spec.containers[name=nginx].image=nginx:1\.7\.9" my-dir/ | kpt cfg tree

###

[tutorial-script]: ../gifs/cfg-grep.sh
