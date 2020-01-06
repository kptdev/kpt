// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmdutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
)

const (
	SilenceErrorsEnv   = "COBRA_SILENCE_USAGE"
	StackTraceOnErrors = "COBRA_STACK_TRACE_ON_ERRORS"
	trueString         = "true"
)

// FixDocs replaces instances of old with new in the docs for c
func FixDocs(old, new string, c *cobra.Command) {
	c.Use = strings.ReplaceAll(c.Use, old, new)
	c.Short = strings.ReplaceAll(c.Short, old, new)
	c.Long = strings.ReplaceAll(c.Long, old, new)
	c.Example = strings.ReplaceAll(c.Example, old, new)
}

func SetSilenceErrors(c *cobra.Command) {
	e := os.Getenv(SilenceErrorsEnv)
	if e == trueString || e == "1" {
		c.SilenceUsage = true
	}
}

func HandlePreRunError(c *cobra.Command, err error) error {
	e := os.Getenv(StackTraceOnErrors)
	if StackOnError || e == trueString || e == "1" {
		if err, ok := err.(*errors.Error); ok {
			fmt.Fprint(os.Stderr, fmt.Sprintf("%s", err.Stack()))
		}
	}
	return err
}

func HandleError(c *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	e := os.Getenv(StackTraceOnErrors)
	if StackOnError || e == trueString || e == "1" {
		if err, ok := err.(*errors.Error); ok {
			fmt.Fprint(os.Stderr, fmt.Sprintf("%s", err.Stack()))
		}
	}

	if ExitOnError {
		fmt.Fprintf(c.ErrOrStderr(), "Error: %v\n", err)
		os.Exit(1)
	}
	return err
}

// ExitOnError if true, will cause commands to call os.Exit instead of returning an error.
// Used for skipping printing usage on failure.
var ExitOnError bool

// StackOnError if true, will print a stack trace on failure.
var StackOnError bool
