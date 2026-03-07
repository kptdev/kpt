# End-to-end testing of kpt

## kpt live e2e tests

We currently have two different solutions for running e2e tests for the kpt
live functionality. We are working on reconciling this into one approach that
we can use consistently.

All e2e tests for live requires that kind is available.

### Testing with go

We have a framework for running e2e tests based on specifying test cases
under testdata/live-apply folder (tests for other kpt live commands will be
added). The entry point for the test framework is in
the `live_test.go` file.

In order to run all the tests for live apply, there is a make target
```sh
make test-live-apply
```

It is possible to run a single test by specifying the name of the test case:
```sh
make test-live-apply T=crd-and-cr
```

#### Structure of a test case

Each test case is a folder directly under `testdata/live-apply`. In the root
of each test case folder there must be a `config.yaml` file that provides the
configuration of the test case (like whether a clean cluster is required and
the expected output). The package that will be applied with `kpt live apply` is
provided in the `resources` folder.

#### Configuration options

These are the configuration options available in the `config.yaml` file:
 * `exitCode`: Defines the expected exit code after running the kpt live command. Defaults to 0.
 * `stdErr`: Defines the expected output to stdErr after running the command. Defaults to "".
 * `stdOut`: Defines the expected output to stdOut after running the command. Defaults to "".
 * `inventory`: Defines the expected inventory after running the command.
 * `requiresCleanCluster`: Defines whether a new kind cluster should be created prior to running the test.
 * `preinstallResourceGroup`: Defines whether the framework should make sure the RG CRD is available before running the test.
 * `kptArgs`: Defines the arguments that will be used with executing the kpt live command.

## Testing with bash

This approach uses a bash script that runs through several scenarios for
kpt live in sequence. Run it by running
```sh
./live/end-to-end-test.sh
```