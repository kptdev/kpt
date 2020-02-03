## annotate

Set an annotation on one or more Resources

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg-annotate.cast" speed="1" theme="solarized-dark" cols="60" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    kpt tutorial cfg annotate

[tutorial-script]

### Synopsis

  DIR:
    Path to local directory.

### Examples

    # set an annotation on all Resources: 'key: value'
    kpt cfg annotate DIR --kv key=value

    # set an annotation on all Service Resources
    kpt cfg annotate DIR --kv key=value --kind Service

    # set an annotation on the foo Service Resource only
    kpt cfg annotate DIR --kv key=value --kind Service --name foo

    # set multiple annotations
    kpt cfg annotate DIR --kv key1=value1 --kv key2=value2

### 

