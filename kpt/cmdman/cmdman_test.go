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

package cmdman_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kpt.dev/kpt/cmdman"
	"lib.kpt.dev/kptfile"
)

// TestCmd_execute tests that the command displays package documentation.
func TestCmd_execute(t *testing.T) {
	d, err := ioutil.TempDir("", "kptman")
	assert.NoError(t, err)

	// write the KptFile
	err = ioutil.WriteFile(filepath.Join(d, kptfile.KptFileName), []byte(`
apiVersion: kpt.dev/v1alpha1
kind: KptFile
metadata:
  name: java
packageMetadata:
  man: "man/MAN.md"
`), 0600)
	assert.NoError(t, err)

	// write the man md file
	err = os.Mkdir(filepath.Join(d, "man"), 0700)
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(d, "man", "MAN.md"), []byte(`
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
	c := cmdman.NewRunner()
	c.Command.SetArgs([]string{d})
	c.Man.ManExecCommand = "cat"
	c.Command.SetOut(b)
	err = c.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, `.TH java 1   "June 2019"  "Application"

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
  name: java\-config

.fi
.RE
`, b.String())
}

func TestCmd_execute_manNotSpecified(t *testing.T) {
	d, err := ioutil.TempDir("", "kptman")
	assert.NoError(t, err)

	if !assert.NoError(t, os.Mkdir(filepath.Join(d, "pkg"), 0700)) {
		return
	}
	if !assert.NoError(t, os.Chdir(d)) {
		return
	}

	// write the KptFile
	err = ioutil.WriteFile(filepath.Join(d, "pkg", kptfile.KptFileName), []byte(`
apiVersion: kpt.dev/v1alpha1
kind: KptFile
metadata:
  name: java
`), 0600)
	assert.NoError(t, err)

	// write the man md file
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(d, "pkg", "MAN.md"), []byte(`
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
	c := cmdman.NewRunner()
	c.Command.SetArgs([]string{"pkg"})
	c.Man.ManExecCommand = "cat"
	c.Command.SetOut(b)
	err = c.Command.Execute()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `.TH java 1   "June 2019"  "Application"

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
  name: java\-config

.fi
.RE
`, b.String())
}

// TestCmd_fail verifies that command returns an error if the directory is
// not found.
func TestCmd_fail(t *testing.T) {
	r := cmdman.NewRunner()
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Man.ManExecCommand = "cat"
	r.Command.SetArgs([]string{filepath.Join("not", "real", "dir")})
	err := r.Command.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestCmd_defaultDir verifies that '.' is used as the default dir if none
// is provided as an argument.
func TestCmd_defaultDir(t *testing.T) {
	r := cmdman.NewRunner()
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Man.ManExecCommand = "cat"
	err := r.Command.Execute()
	assert.EqualError(t, err, fmt.Sprintf(
		"unable to read %s: open %s: no such file or directory",
		kptfile.KptFileName, kptfile.KptFileName))
}
