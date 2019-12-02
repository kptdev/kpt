## kpt

  Manage configuration packages using git

### Synopsis

  Manage configuration packages using git.

  get, update, and publish packages on Resource configuration as subdirectories of
  git repositories.

  For best results, use with tools such as kustomize and kubectl.

    kpt SUB_CMD [flags]

#### Flags

  --stack-trace
  
    Print a stack trace on an error

#### Env Vars

  COBRA_SILENCE_ERRORS
  
    Set to true to silence printing the usage on error
    
  COBRA_STACK_TRACE_ON_ERRORS
  
    Set to true to print a stack trace on an error

### Examples

    # view kpt subcommands
    kpt help
    
    # print the tutorial for fetching a package
    kpt help tutorials-1-get
    
    # read the documentation on the get command
    kpt help get
