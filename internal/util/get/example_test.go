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

package get_test

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/util/get"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
)

func ExampleCommand() {
	err := get.Command{Git: &kptfilev1.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "v1.0",
	}}.Run(fake.CtxWithDefaultPrinter())
	if err != nil {
		fmt.Print(err.Error())
	}
}

func ExampleCommand_branch() {
	err := get.Command{Git: &kptfilev1.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "refs/heads/v1.0",
	}}.Run(fake.CtxWithDefaultPrinter())
	if err != nil {
		fmt.Print(err.Error())
	}
}

func ExampleCommand_tag() {
	err := get.Command{Git: &kptfilev1.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "refs/tags/v1.0",
	}}.Run(fake.CtxWithDefaultPrinter())
	if err != nil {
		fmt.Print(err.Error())
	}
}

func ExampleCommand_commit() {
	err := get.Command{Git: &kptfilev1.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "8186bef8e5c0621bf80fa8106bd595aae8b62884",
	}}.Run(fake.CtxWithDefaultPrinter())
	if err != nil {
		fmt.Print(err.Error())
	}
}

func ExampleCommand_subdir() {
	err := get.Command{
		Git: &kptfilev1.Git{
			Repo:      "https://github.com/example-org/example-repo",
			Ref:       "v1.0",
			Directory: filepath.Join("path", "to", "package"),
		},
	}.Run(fake.CtxWithDefaultPrinter())
	if err != nil {
		fmt.Print(err.Error())
	}
}

func ExampleCommand_destination() {
	err := get.Command{
		Git: &kptfilev1.Git{
			Repo: "https://github.com/example-org/example-repo",
			Ref:  "v1.0",
		},
		Destination: "destination-dir"}.Run(fake.CtxWithDefaultPrinter())
	if err != nil {
		fmt.Print(err.Error())
	}
}
