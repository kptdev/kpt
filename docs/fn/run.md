## run

Run a function locally against Resource configuration.

### Synopsis

`run` sequentially locally executes one or more programs which may modify Resource configuration.

#### Architecture

- Programs are packaged as container images which are pulled and run locally
- Input Resource configuration is read from some _source_ and written to container `STDIN`
- Output Resource configuration is read from container `STDOUT` and written to some _sink_
- If the container exits non-0, run will fail and print the container `STDERR`

#### Invocations Patterns

1. Run an individual function explicitly

    A function may be explicitly specified with `--image`

    Example: Locally run the container image gcr.io/example.com/my-fn against the Resources
    in DIR/:

         kpt fn run DIR/ --image gcr.io/example.com/my-fn

    If not `DIR` is specified, the source will default to STDIN and sink will default to STDOUT. For Example, this is equivalent to the preceding example.

         kpt source DIR/ | kpt fn run --image gcr.io/example.com/my-fn | kpt sink DIR/

    Arguments specified after `--` will be provided to the function as a ConfigMap input

    Example: In addition to the input Resources, provide to the container image a ConfigMap
    containing `data: {foo: bar}`. This is used to set the behavior of the function.

           # run the my-fn image, configured with foo=bar
           kpt fn run DIR/ --image gcr.io/example.com/my-fn:v1.0.0 -- foo=bar

2. Declare one or more functions to be run

    Functions and their input configuration may be declared in files rather than directly on the command line.

    For example, this is equivalent to the preceding example.
    Rather than specifying `gcr.io/example.com/my-fn` as a flag, specify it in a file using the
    `config.kubernetes.io/function` annotation.

        # run the my-fn, configured with foo=bar -- fn is declared in the input
        kpt fn run DIR/

        # DIR/some.yaml
        apiVersion: v1
        kind: ConfigMap
        metdata:
          annotations:
            config.kubernetes.io/function: |
              container:
                image: gcr.io/example.com/my-fn
        data:
          foo: bar

    Functions which are nested under some sub directory are scoped only to Resources under that
    same sub directory. This allows fine grain control over how functions are executed:

        Example: gcr.io/example.com/my-fn is scoped to Resources in stuff/ and
        is NOT scoped to Resources in apps/
             .
             ├── stuff
             │   ├── inscope-deployment.yaml
             │   ├── stuff2
             │   │     └── inscope-deployment.yaml
             │   └── my-function.yaml # functions is scoped to stuff/...
             └── apps
                 ├── not-inscope-deployment.yaml
                 └── not-inscope-service.yaml

    Alternatively, you can also place function configurations in a special directory named `functions`. For example:
    this is equivalent to previous example:

        Example: gcr.io/example.com/my-fn is scoped to Resources in stuff/ and
        is NOT scoped to Resources in apps/
             .
             ├── stuff
             │   ├── inscope-deployment.yaml
             │   ├── stuff2
             │   │     └── inscope-deployment.yaml
             │   └── functions
             │         └── my-function.yaml
             └── apps
                 ├── not-inscope-deployment.yaml
                 └── not-inscope-service.yaml

    Alternatively, you can also use `--fn-path` to explicitly provide the directory containing function configurations:

        # run the my-fn, configured with foo=bar -- fn is declared in a file
        kpt fn run DIR/ --fn-path FUNCTIONS_DIR/

        # FUNCTIONS_DIRS/some.yaml
        apiVersion: v1
        kind: ConfigMap
        metdata:
          annotations:
            config.kubernetes.io/function: |
              container:
                image: gcr.io/example.com/my-fn
        data:
          foo: bar

    You may declare multiple functions. If they are specified in the same file they will
    be run in the same order that they are specified.

    Functions may define their own API input types - these may be client-side equivalents of CRDs.

          kpt fn run DIR/

          # DIR/functions/some.yaml
          apiVersion: v1
          kind: ConfigMap
          metdata:
            annotations:
              config.kubernetes.io/function: |
                container:
                  image: gcr.io/example.com/my-fn
          data:
            foo: bar
          ---
          apiVersion: v1
          kind: MyType
          metdata:
            annotations:
              config.kubernetes.io/function: |
                container:
                  image: gcr.io/example.com/other-fn
          spec:
            field:
              nestedField: value

#### Developing functions

There are several projects that may be used to quickly develop `kpt functions`:

- Typescript: [kpt functions sdk](https://github.com/GoogleContainerTools/kpt-functions-sdk)
- Golang: [kyaml](https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml)

### Examples

    # read the Resources from DIR, provide them to a container my-fun as input,
    # write my-fn output back to DIR
    kpt fn run DIR/ --image gcr.io/example.com/my-fn

    # provide the my-fn with an input ConfigMap containing `data: {foo: bar}`
    kpt fn run DIR/ --image gcr.io/example.com/my-fn:v1.0.0 -- foo=bar

    # run the functions in FUNCTIONS_DIR against the Resources in DIR
    kpt fn run DIR/ --fn-path FUNCTIONS_DIR/

    # discover functions in DIR and run them against Resource in DIR.
    # functions may be scoped to a subset of Resources -- see `kpt help fn run`
    kpt fn run DIR/
