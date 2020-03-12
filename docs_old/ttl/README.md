## ttl

Run command tutorials

### Synopsis

Tutorials simulates running a sequence of commands and their output by playing
an asciinema cast.

Requires [asciinema].

### Examples

    # run the tutorial for `kpt cfg annotate`
    kpt ttl cfg annotate

    # run the tutorial at 2x speed
    kpt ttl cfg annotate -s 2

    # run the tutorial at 1/2x speed
    kpt ttl cfg annotate -s 0.5

    # print the full tutorial rather than playing it
    kpt ttl cfg annotate --print

###

[asciinema]: https://asciinema.org/docs/usage
