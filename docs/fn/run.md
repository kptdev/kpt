## run

Locally executes one or more programs which may generate, transform, or validate configuration files.

### Synopsis

Programs are packaged as container images which are pulled and run locally.
If the container exits with non-zero status code, run will fail and print the
container `STDERR`.

#### Imperatively run an single function

A function may be explicitly specified using the `--image` flag.

__Example:__ Locally run the container image `gcr.io/example.com/my-fn` against
the Resources in `DIR/`:

```sh
kpt fn run DIR/ --image gcr.io/example.com/my-fn
```

If `DIR/` is not specified, the source will default to STDIN and sink will default
to STDOUT.

__Example:__ this is equivalent to the preceding example:

```sh
kpt source DIR/ |
kpt fn run --image gcr.io/example.com/my-fn |
kpt sink DIR/
```

Arguments specified after `--` will be provided to the function as a `ConfigMap` input.

__Example:__ In addition to the input Resources, provide to the container image a
`ConfigMap` containing `data: {foo: bar}`. This is used to parameterize the behavior
of the function:

```sh
kpt fn run DIR/ --image gcr.io/example.com/my-fn -- foo=bar
```

#### Declaratively run one or more functions

Functions and their input configuration may be declared in files rather than directly
on the command line.

__Example:__ This is equivalent to the preceding example:

Create a file e.g. `DIR/my-function.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metdata:
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/example.com/my-fn
data:
  foo: bar
```

Run the function:

``` sh
kpt fn run DIR/
```

Here, rather than specifying `gcr.io/example.com/my-fn` as a flag, we specify it in a
file using the `config.kubernetes.io/function` annotation.

##### Scoping Rules

Functions which are nested under some sub directory are scoped only to Resources under 
that same sub directory. This allows fine grain control over how functions are 
executed:

__Example:__ Function declared in `stuff/my-function.yaml` is scoped to Resources in 
`stuff/` and is NOT scoped to Resources in `apps/`:

```sh
.
├── stuff
│   ├── inscope-deployment.yaml
│   ├── stuff2
│   │     └── inscope-deployment.yaml
│   └── my-function.yaml # functions is scoped to stuff/...
└── apps
    ├── not-inscope-deployment.yaml
    └── not-inscope-service.yaml
```

Alternatively, you can also place function configurations in a special directory named 
`functions`.

__Example__: This is equivalent to previous example:

```sh
.
├── stuff
│   ├── inscope-deployment.yaml
│   ├── stuff2
│   │     └── inscope-deployment.yaml
│   └── functions
│         └── my-function.yaml
└── apps
    ├── not-inscope-deployment.yaml
    └── not-inscope-service.yaml
```

Alternatively, you can also use `--fn-path` to explicitly provide the directory 
containing function configurations:

```sh
kpt fn run DIR/ --fn-path FUNCTIONS_DIR/
```

Alternatively, scoping can be disabled using `--global-scope` flag.

##### Declaring Multiple Functions

You may declare multiple functions. If they are specified in the same file 
(multi-object YAML file separated by `---`), they will
be run sequentially in the order that they are specified.

##### Custom `functionConfig`

Functions may define their own API input types - these may be client-side equivalents 
of CRDs:

__Example__: Declare two functions in `DIR/functions/my-functions.yaml`:

```yaml
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
```

### Examples

```sh
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
```

