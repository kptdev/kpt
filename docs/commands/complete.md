## install-completion

Install shell completion for kpt commands and flags

### Synopsis

Install shell completion for kpt commands and flags.

    kpt install-completion

Uninstall shell completion.

    COMP_UNINSTALL=1 kpt complete

### Examples

    # install
    $ kpt install-completion
    install completion for kpt? y
    $ source ~/.bash_profile

    # uninstall
    $ COMP_UNINSTALL=1 kpt install-completion
    uninstall completion for kpt? y 
    $ source ~/.bash_profile
