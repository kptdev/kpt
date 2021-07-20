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

// Package man contains libraries for rendering package documentation as man
// pages.
package man

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/cpuguy83/go-md2man/v2/md2man"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// Command displays local package documentation as man pages.
// The location of the man page is read from the Kptfile packageMetadata.
// If no man page is specified, and error is returned.
//
// Man page format should be the format supported by the
// github.com/cpuguy83/go-md2man/md2man library
type Command struct {
	// Path is the path to a local package
	Path string

	// ManExecCommand is the exec command to run for displaying the man pages.
	ManExecCommand string

	// StdOut is the StdOut value
	StdOut io.Writer
}

const ManFilename = "README.md"

// Run runs the command.
func (m Command) Run() error {
	_, err := exec.LookPath(m.GetExecCmd())
	if err != nil {
		return errors.Errorf(m.GetExecCmd() + " not installed")
	}

	// lookup the path to the man page
	k, err := pkg.ReadKptfile(m.Path)
	if err != nil {
		return err
	}
	if k.Info == nil {
		k.Info = &v1.PackageInfo{}
	}

	if k.Info.Man == "" {
		_, err := os.Stat(filepath.Join(m.Path, ManFilename))
		if err != nil {
			return errors.Errorf("no manual entry for %q", m.Path)
		}
		k.Info.Man = filepath.Join(ManFilename)
	}

	// the path separator needs to converted to the OS path.
	// e.g. the separator will be different for windows.
	p := filepath.Join(path.Split(k.Info.Man))

	// verify the man page is in the package
	apPkg, err := filepath.Abs(m.Path)
	if err != nil {
		return err
	}
	apMan, err := filepath.Abs(filepath.Join(m.Path, p))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(apMan, apPkg) {
		return errors.Errorf("invalid manual location for %q", m.Path)
	}

	// write the formatted manual to a tmp file so it can be displayed
	f, err := ioutil.TempFile("", "kpt-man")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	b, err := ioutil.ReadFile(apMan)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(f.Name(), md2man.Render(b), 0600)
	if err != nil {
		return err
	}

	// setup the man command
	manCmd := exec.Command(m.GetExecCmd(), f.Name())
	manCmd.Stderr = os.Stderr
	manCmd.Stdin = os.Stdin
	manCmd.Stdout = m.GetStdOut()
	manCmd.Env = os.Environ()
	return manCmd.Run()
}

// GetExecCmd returns the command that will be executed to display the
// man pages.
func (m Command) GetExecCmd() string {
	if m.ManExecCommand == "" {
		return "man"
	}
	return m.ManExecCommand
}

// GetStdOut returns the io.Writer that will be used as the man stdout
func (m Command) GetStdOut() io.Writer {
	if m.StdOut == nil {
		return os.Stdout
	}
	return m.StdOut
}
