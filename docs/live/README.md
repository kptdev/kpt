## live

Reconcile configuration files with the live state

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="coming..." speed="1" theme="solarized-dark" cols="60" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    # run the tutorial from the cli
    kpt tutorial live

[tutorial-script]

### Synopsis

Tool to safely apply and delete kubernetes package resources from live clusters.

| Command   | Description                                               |
|-----------|-----------------------------------------------------------|
| [apply]   | apply a package to the live cluster                       |
| [preview] | preview the operations that apply or destroy will perform |
| [destroy] | delete the package resources from the live cluster        |

**Data Flow**: local configuration or stdin -> kpt live -> apiserver (Kubernetes cluster)

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | apiserver                |
| apiserver               | stdout                   |

[tutorial-script]: ../gifs/live.sh
[apply]: apply.md
[preview]: preview.md
[destroy]: destroy.md
