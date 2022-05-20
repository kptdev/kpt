This package is an end-to-end test runner for kpt fn commands. It
can also be used to test kpt KRM functions.

# How Does It Work

After importing this package, you can use `runner.ScanTestCases` to
recursively scan the test cases in a given directory. For each test
case, call `runner.NewRunner` to create a test runner and then call
`Run` to run the test. For example please see `e2e/fn_test.go`

All test cases should be **independent** and should be able to run in
fully parallel. They should be **self-contained** so the runner doesn't
need any manual configuration.

The configurations for a test case is located in the `.expected` directory
in the test case directory. The runner will implicitly use following files
in `.expected`:

- `config.yaml`: This is the file which contains the configurations for the
  test case. It can have following fields:
  - `testType`: `eval` or `render`. It controls which `kpt fn` command will be
    used to run the test. Default: `render`.
  - `exitCode`: The expected exit code for the command. Default: 0.
  - `skip`: Runner will skip the test if `skip` is set to true. Default: false.
  - `sequential`: This test case should be run sequentially. Default: false.
  - `runtimes`: If the current runtime doesn't match any of the desired runtimes
    here, the test case will be skipped. Valid values are `docker` and `podman`.
    If unspecified, it will match any runtime.
  - `imagePullPolicy`: The image pull policy to be used. It can be set to one of
    `Always`, `IfNotPresent` and `Never`. Default value is inherited from the
    CLI flag.
  - `notIdempotent`: The functions and commands should be idempotent, but in
    some cases it's not doable. The runner will not run the command twice to
    check idempotent if this is set to true. Default: false.
  - `debug`: Debug means will the debug behavior be enabled. Default: false.
    Debug behaviors:
    - Keep the temporary directory used to run the test cases after test.
  - `stdOut`: The expected standard output from running the command. Default: "".
  - `stdErr`: The expected standard error from running the command. Default: "".
  - `StdErrRegEx`: A regular expression that is expected to match the standard error. Default: "".
  - `disableOutputTruncate`: Control should error output be truncated. Default:
    false.
  - Configurations only apply to `eval` tests:
    - `execPath`: A path to the executable file that will be run as function.
      Mutually exclusive with Image. The path should be separated by slash '/'
    - `image`: The image name for the function. Either `image` or `execPath`
      must be provided.
    - `args`: A map contains the arguments that will be passed into function.
      Args will be passed as 'key=value' format after the '--' in command.
    - `network`: Control the accessibility of network from the function
      container. Default: false.
    - `includeMetaResources`: Enable including meta resources, like Kptfile,
      in the function input. Default: false.
    - `fnConfig`: The path to the function config file. The path should be
      separated by slash '/'.
- `diff.patch`: The expected `git diff` output after running the command.
  Default: "".
- `results.yaml`: The expected result file after running the command.
  Default: "".
- `setup.sh`: A **bash** script which will be run before the command if it exists.
- `exec.sh`: A **bash** script which will be run if it exists and will replace the
  command (`kpt fn eval` or `kpt fn render`) that will be run according to
  `testType` in configurations. All configurations that used to control command
  behavior, like `disableOutputTruncate` and `args`, will be ignored.
- `teardown.sh`: A **bash** script which will be run after the command and
  result comparison if it exists.

# Expected Output

The runner will compare the actual values of:

- `stdout`
- `stderr`
- Exit Code
- Diff
- Results file
