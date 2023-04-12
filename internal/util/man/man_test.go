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

package man_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/util/man"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
)

// TestMan_Execute verifies that Execute will find the man page file,
// format it as a man page, and execute a command to display it.
func TestMan_Execute(t *testing.T) {
	d := t.TempDir()

	// write the KptFile
	err := os.WriteFile(filepath.Join(d, kptfilev1.KptFileName), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: java
info:
  man: "man/README.md"
`), 0600)
	assert.NoError(t, err)

	// write the man md file
	err = os.Mkdir(filepath.Join(d, "man"), 0700)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "man", ManFilename), []byte(`
java 1   "June 2019"  "Application"
==================================================

# NAME
  **java**

# SYNOPSIS

kpt clone testdata3/java

# Description

The **java** package runs a container containing a java application.

# Components

Java server Deployment.

    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: java

Java server Service

    apiVersion: v1
    kind: Service
    metadata:
      name: java

Java server ConfigMap

    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: java-config
`), 0600)
	assert.NoError(t, err)

	b := &bytes.Buffer{}
	instance := Command{
		ManExecCommand: "cat",
		Path:           d,
		StdOut:         b,
	}
	err = instance.Run()
	assert.NoError(t, err)

	assert.Equal(t, `.nh
.TH java 1   "June 2019"  "Application"

.SH NAME
.PP
\fBjava\fP


.SH SYNOPSIS
.PP
kpt clone testdata3/java


.SH Description
.PP
The \fBjava\fP package runs a container containing a java application.


.SH Components
.PP
Java server Deployment.

.PP
.RS

.nf
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java

.fi
.RE

.PP
Java server Service

.PP
.RS

.nf
apiVersion: v1
kind: Service
metadata:
  name: java

.fi
.RE

.PP
Java server ConfigMap

.PP
.RS

.nf
apiVersion: v1
kind: ConfigMap
metadata:
  name: java-config

.fi
.RE
`, b.String())
}

// TestMan_GetExecCmd tests that the exec command is defaulted to "man",
// but can be overridden
func TestMan_GetExecCmd(t *testing.T) {
	// default to "man"
	instance := Command{}
	assert.Equal(t, "man", instance.GetExecCmd())

	// allow overrides for testing
	instance = Command{ManExecCommand: "cat"}
	assert.Equal(t, "cat", instance.GetExecCmd())
}

// TestMan_GetStdOut tests that the command stdout is defaulted to "os.Stdout",
// but can be overridden.
func TestMan_GetStdOut(t *testing.T) {
	// default to stdout
	instance := Command{}
	assert.Equal(t, os.Stdout, instance.GetStdOut())

	// allow overrides for testing
	b := &bytes.Buffer{}
	instance = Command{StdOut: b}
	assert.Equal(t, b, instance.GetStdOut())
}

// TestMan_Execute_failNoManPage verifies that if the man page is not
// specified for the package, an error is returned.
func TestMan_Execute_failNoManPage(t *testing.T) {
	d := t.TempDir()

	// write the KptFile
	err := os.WriteFile(filepath.Join(d, kptfilev1.KptFileName), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: java
info:
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	b := &bytes.Buffer{}
	instance := Command{
		ManExecCommand: "cat",
		Path:           d,
		StdOut:         b,
	}
	err = instance.Run()
	if !assert.EqualError(t, err, fmt.Sprintf("no manual entry for %q", d)) {
		return
	}
	if !assert.Equal(t, ``, b.String()) {
		return
	}
}

// TestMan_Execute_failBadPath verifies that Execute will fail if the man
// path does not exist.
func TestMan_Execute_failBadPath(t *testing.T) {
	d := t.TempDir()

	// write the KptFile
	err := os.WriteFile(filepath.Join(d, kptfilev1.KptFileName), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: java
info:
  man: "not/real/path"
`), 0600)
	assert.NoError(t, err)

	b := &bytes.Buffer{}
	instance := Command{
		ManExecCommand: "cat",
		Path:           d,
		StdOut:         b,
	}
	err = instance.Run()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
	assert.Equal(t, ``, b.String())
}

// TestMan_Execute_failLocation verifies that Execute will fail if the man
// path is not under the package directory.
func TestMan_Execute_failLocation(t *testing.T) {
	d := t.TempDir()

	// write the KptFile
	err := os.WriteFile(filepath.Join(d, kptfilev1.KptFileName), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: java
info:
  man: "../../path"
`), 0600)
	assert.NoError(t, err)

	b := &bytes.Buffer{}
	instance := Command{
		ManExecCommand: "cat",
		Path:           d,
		StdOut:         b,
	}
	err = instance.Run()
	assert.EqualError(t, err, fmt.Sprintf("invalid manual location for %q", d))
	assert.Equal(t, ``, b.String())
}

// TestMan_Execute_failLocation verifies that Execute will fail if the man
// path is not under the package directory.
func TestMan_Execute_failManNotInstalled(t *testing.T) {
	b := &bytes.Buffer{}
	instance := Command{
		ManExecCommand: "notrealprogram",
		Path:           "path",
		StdOut:         b,
	}
	err := instance.Run()
	assert.EqualError(t, err, "notrealprogram not installed")
	assert.Equal(t, ``, b.String())
}
