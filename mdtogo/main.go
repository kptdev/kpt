// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// Package main generates cobra.Command go variables containing documentation read from .md files.
// Usage: mdtogo SOURCE_MD_DIR/ DEST_GO_DIR/ [--recursive=true] [--license=license.txt|none]
//
// The command will create a docs.go file under DEST_GO_DIR/ containing string variables to be
// used by cobra commands for documentation. The variable names are generated from the name of
// the directory in which the files resides, replacing '-' with ”, title casing the name.
// All *.md files will be read from DEST_GO_DIR/, including subdirectories if --recursive=true,
// and a single DEST_GO_DIR/docs.go file is generated.
//
// The content for each of the three variables created per folder, are set
// by looking for a HTML comment on one of two forms:
//
// <!--mdtogo:<VARIABLE_NAME>-->
//
//	..some content..
//
// <!--mdtogo-->
//
// or
//
// <!--mdtogo:<VARIABLE_NAME>
// ..some content..
// -->
//
// The first are for content that should show up in the rendered HTML, while
// the second is for content that should be hidden in the rendered HTML.
//
// <VARIABLE_NAME> must be suffixed with Short, Long or Examples; <VARIABLE_NAME>s without
// a prefix will have an assumed prefix of the parent directory of the markdown file.
//
// Flags:
//
//	--recursive=true
//	  Scan the directory structure recursively for .md files
//	--license
//	  Controls the license header added to the files.  Specify a path to a license file,
//	  or "none" to skip adding a license.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/mdtogo/cmddocs"
	"github.com/GoogleContainerTools/kpt/mdtogo/common"
)

var recursive bool
var licenseFile string
var strategy string

const (
	cmdDocsStrategy = "cmdDocs"
	futureStrategy  = "future" // please replace it with the next strategy we add
)

func main() {
	for _, a := range os.Args {
		if a == "--recursive=true" {
			recursive = true
		}
		if strings.HasPrefix(a, "--strategy=") {
			switch a {
			case "--strategy=cmdDocs":
				strategy = cmdDocsStrategy
			default:
				fmt.Fprintf(os.Stderr, "Invalid strategy %s\n", a)
				os.Exit(1)
			}
		}
		if strings.HasPrefix(a, "--license=") {
			licenseFile = strings.ReplaceAll(a, "--license=", "")
		}
	}

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: mdtogo SOURCE_MD_DIR/ DEST_GO_DIR/\n")
		os.Exit(1)
	}
	source := os.Args[1]
	dest := os.Args[2]

	files, err := common.ReadFiles(source, recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	license := getLicense()

	switch strategy {
	case cmdDocsStrategy:
		docs := cmddocs.ParseCmdDocs(files)
		err = cmddocs.Write(docs, dest, license)
	case futureStrategy:
		err = errors.New("this strategy should not be used, please replace it with a real strategy")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func getLicense() string {
	var license string

	switch licenseFile {
	case "":
		license = `// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0`
	case "none":
		// no license -- maybe added by another tool
	default:
		b, err := os.ReadFile(licenseFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		license = string(b)
	}
	return license
}
