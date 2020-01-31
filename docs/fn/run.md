## run

Run a function locally against Resource configuration.

### Synopsis

`run` sequentially locally executes one or more programs which may modify Resource configuration.

**Architecture:**

- Programs are packaged as container images which are pulled and run locally
- Input Resource configuration is read from some *source* and written to container `STDIN`
- Output Resource configuration is read from container `STDOUT` and written to some *sink*
- If the container exits non-0, run will fail and print the container `STDERR`

**Caveats:**

- If `DIR` is provided as an argument, it is used as both the *source* and *sink*.
- A function may be explicitly specified with `--image`

  Example: Locally run the container image gcr.io/example.com/my-fn against the Resources
  in DIR/.


        kpt fn run DIR/ --image gcr.io/example.com/my-fn

- If not `DIR` is specified, the source will default to STDIN and sink will default to STDOUT

  Example: This is equivalent to the preceding example.


        kpt source DIR/ | kpt fn run --image gcr.io/example.com/my-fn | kpt sink DIR/

- Arguments specified after `--` will be provided to the function as a ConfigMap input 

  Example: In addition to the input Resources, provide to the container image a ConfigMap
  containing `data: {foo: bar}`.  This is used to set the behavior of the function.


        # run the my-fn image, configured with foo=bar
        kpt fn run DIR/ --image gcr.io/example.com/my-fn:v1.0.0 -- foo=bar

- Alternatively `functions` and their input configuration may be declared in
  files rather than directly on the command line
- `FUNCTIONS_DIR` may optionally be under the Resource `DIR`

  Example: This is equivalent to the preceding example.
  Rather than specifying `gcr.io/example.com/my-fn` as a flag, specify it in a file using the
  `config.kubernetes.io/function` annotation, and discover it with `--fn-path`.


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

- Additionally, `functions` may be discovered implicitly by putting them in `run` *source*.


        # run the my-fn, configured with foo=bar -- fn is declared in the input
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

- `functions` which are nested under some sub directory are scoped only to Resources under that
  same sub directory.  This allows fine grain control over how `functions` are executed.


         Example: gcr.io/example.com/my-fn is scoped to Resources in stuff/ and
         is NOT scoped to Resources in apps/
             .
             ├── stuff
             │   ├── inscope-deployment.yaml
             │   ├── stuff2
             │   │     └── inscope-deployment.yaml
             │   └── functions # functions in this dir are scoped to stuff/...
             │       └── some.yaml
             └── apps
                 ├── not-inscope-deployment.yaml
                 └── not-inscope-service.yaml

- Multiple `functions` may be specified.  If they are specified in the same file they will
  be run in the same order that they are specified.
- `functions` may define their own API input types - these may be client-side equivalents of CRDs.


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

#### Arguments:

  DIR:
    Path to local directory.

#### KPT Functions:

  `kpt functions` are specified as Kubernetes types containing a `metadata.annotations.[config.kubernetes.io/function]`
  field specifying an image for the container to run.  This image tells run how to invoke the container.

  Example kpt function:

	# in file example/fn.yaml
	apiVersion: fn.example.com/v1beta1
	kind: ExampleFunctionKind
	metadata:
	  annotations:
	    config.kubernetes.io/function: |
	      container:
	        # function is invoked as a container running this image
	        image: gcr.io/example/examplefunction:v1.0.1
	    config.kubernetes.io/local-config: "true" # tools should ignore this
	spec:
	  configField: configValue

  In the preceding example, 'kpt cfg run example/' would identify the function by
  the `metadata.annotations.[config.kubernetes.io/function]` field.  It would then write all Resources in the directory to
  a container stdin (running the `gcr.io/example/examplefunction:v1.0.1` image).  It
  would then write the container stdout back to example/, replacing the directory
  file contents.

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
