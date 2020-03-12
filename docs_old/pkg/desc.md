## desc

Print the package origin

### Synopsis

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/pkg-desc.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

`desc` reads package information in given DIRs and displays it in tabular format.
Input can be a list of package directories (defaults to the current directory if not specifed).
Any directory with a Kptfile is considered to be a package.

    kpt pkg desc [DIR]...

### Examples

    # display description for package in current directory
    kpt pkg desc

    # display description for packages in directories with 'prod-' prefix
    kpt pkg desc prod-*

