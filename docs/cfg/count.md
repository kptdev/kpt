## count

Print resource counts

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg-count.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    kpt tutorial cfg count

[tutorial-script]

### Synopsis

    kpt cfg count [DIR]

  DIR:
    Path to local directory.

### Examples

    # print Resource counts from a directory
    kpt cfg count my-dir/

    # print Resource counts from a cluster
    kubectl get all -o yaml | kpt cfg count

### 
