// Copyright 2019 The kpt Authors
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

package fix

import (
	"fmt"
	"io"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/fix/fixsetters"
)

// Command fixes the local kpt package and upgrades it to use latest feature
type Command struct {
	// PkgPath path to the kpt package directory
	PkgPath string

	// DryRun indicates that only preview of actions should be printed without
	// performing actual actions
	DryRun bool

	// StdOut standard out to write messages to
	StdOut io.Writer
}

// Run runs the Command.
func (c Command) Run() error {
	printFunc := printFunc(c.StdOut, c.DryRun)
	printFunc("processing resource configs to identify possible fixes... ")
	return c.fixV1Setters()
}

func (c Command) fixV1Setters() error {
	printFunc := printFunc(c.StdOut, c.DryRun)
	f := &fixsetters.SetterFixer{
		PkgPath:     c.PkgPath,
		OpenAPIPath: filepath.Join(c.PkgPath, "Kptfile"),
		DryRun:      c.DryRun,
	}
	sfr, err := f.FixV1Setters()
	if err != nil {
		return err
	}
	if !sfr.NeedFix {
		printFunc("package is using latest version of setters, no fix needed")
		return nil
	}

	for _, setter := range sfr.CreatedSetters {
		printFunc("created setter with name %s", setter)
	}
	printFunc("created %d setters in total", len(sfr.CreatedSetters))

	for _, subst := range sfr.CreatedSubst {
		printFunc("created substitution with name %s", subst)
	}
	printFunc("created %d substitution in total", len(sfr.CreatedSubst))

	for setter, err := range sfr.FailedSetters {
		printFunc("failed to create setter with name %s: %v", setter, err)
	}

	for subst, err := range sfr.FailedSubst {
		printFunc("failed to create substitution with name %s: %v", subst, err)
	}

	return err
}

type printerFunc func(format string, a ...interface{})

func printFunc(w io.Writer, dryRun bool) printerFunc {
	return func(format string, a ...interface{}) {
		if dryRun {
			format += " (dry-run)"
		}
		fmt.Fprintf(w, format+"\n", a...)
	}
}
