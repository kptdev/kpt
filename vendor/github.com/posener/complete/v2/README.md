# complete

[![Build Status](https://travis-ci.org/posener/complete.svg?branch=master)](https://travis-ci.org/posener/complete)
[![codecov](https://codecov.io/gh/posener/complete/branch/master/graph/badge.svg)](https://codecov.io/gh/posener/complete)
[![golangci](https://golangci.com/badges/github.com/posener/complete.svg)](https://golangci.com/r/github.com/posener/complete)
[![GoDoc](https://godoc.org/github.com/posener/complete?status.svg)](https://pkg.go.dev/github.com/posener/complete/v2?tab=doc)
[![goreadme](https://goreadme.herokuapp.com/badge/posener/complete.svg)](https://goreadme.herokuapp.com)

Package complete is everything for bash completion and Go.

Writing bash completion scripts is a hard work, usually done in the bash scripting language.
This package provides:

* A library for bash completion for Go programs.

* A tool for writing bash completion script in the Go language. For any Go or non Go program.

* Bash completion for the `go` command line (See [./gocomplete](./gocomplete)).

* Library for bash-completion enabled flags (See [./compflag](./compflag)).

* Enables an easy way to install/uninstall the completion of the command.

The library and tools are extensible such that any program can add its one logic, completion types
or methologies.

#### Go Command Bash Completion

[./gocomplete](./gocomplete) is the script for bash completion for the `go` command line. This is an example
that uses the `complete` package on the `go` command - the `complete` package can also be used to
implement any completions, see #usage.

Install:

1. Type in your shell:

```go
go get -u github.com/posener/complete/v2/gocomplete
COMP_INSTALL=1 gocomplete
```

2. Restart your shell

Uninstall by `COMP_UNINSTALL=1 gocomplete`

Features:

- Complete `go` command, including sub commands and flags.
- Complete packages names or `.go` files when necessary.
- Complete test names after `-run` flag.

#### Complete Package

Supported shells:

- [x] bash
- [x] zsh
- [x] fish

#### Usage

Add bash completion capabilities to any Go program. See [./example/command](./example/command).

```go
 import (
 	"flag"
 	"github.com/posener/complete/v2"
 	"github.com/posener/complete/v2/predict"
 )
 var (
 	// Add variables to the program.
 	name      = flag.String("name", "", "")
 	something = flag.String("something", "", "")
 	nothing   = flag.String("nothing", "", "")
 )
 func main() {
 	// Create the complete command.
 	// Here we define completion values for each flag.
 	cmd := &complete.Command{
	 	Flags: map[string]complete.Predictor{
 			"name":      predict.Set{"foo", "bar", "foo bar"},
 			"something": predict.Something,
 			"nothing":   predict.Nothing,
 		},
 	}
 	// Run the completion - provide it with the binary name.
 	cmd.Complete("my-program")
 	// Parse the flags.
 	flag.Parse()
 	// Program logic...
 }
```

This package also enables to complete flags defined by the standard library `flag` package.
To use this feature, simply call `complete.CommandLine` before `flag.Parse`. (See [./example/stdlib](./example/stdlib)).

```diff
 import (
 	"flag"
+	"github.com/posener/complete/v2"
 )
 var (
 	// Define flags here...
 	foo = flag.Bool("foo", false, "")
 )
 func main() {
 	// Call command line completion before parsing the flags - provide it with the binary name.
+	complete.CommandLine("my-program")
 	flag.Parse()
 }
```

If flag value completion is desired, it can be done by providing the standard library `flag.Var`
function a `flag.Value` that also implements the `complete.Predictor` interface. For standard
flag with values, it is possible to use the `github.com/posener/complete/v2/compflag` package.
(See [./example/compflag](./example/compflag)).

```diff
 import (
 	"flag"
+	"github.com/posener/complete/v2"
+	"github.com/posener/complete/v2/compflag"
 )
 var (
 	// Define flags here...
-	foo = flag.Bool("foo", false, "")
+	foo = compflag.Bool("foo", false, "")
 )
 func main() {
 	// Call command line completion before parsing the flags.
+	complete.CommandLine("my-program")
 	flag.Parse()
 }
```

Instead of calling both `complete.CommandLine` and `flag.Parse`, one can call just `compflag.Parse`
which does them both.

#### Testing

For command line bash completion testing use the `complete.Test` function.

## Sub Packages

* [compflag](./compflag): Package compflag provides a handful of standard library-compatible flags with bash complition capabilities.

* [gocomplete](./gocomplete): Package main is complete tool for the go command line

* [install](./install): Package install provide installation functions of command completion.

* [predict](./predict): Package predict provides helper functions for completion predictors.


---

Created by [goreadme](https://github.com/apps/goreadme)
