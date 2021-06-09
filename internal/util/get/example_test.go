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

package get_test

import (
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
)

func ExampleCommand() {
	err := get.Command{Git: &kptfilev1alpha2.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "v1.0",
	}}.Run(fake.CtxWithEmptyPrinter())
	if err != nil {
		// handle error
	}
}

func ExampleCommand_branch() {
	err := get.Command{Git: &kptfilev1alpha2.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "refs/heads/v1.0",
	}}.Run(fake.CtxWithEmptyPrinter())
	if err != nil {
		// handle error
	}
}

func ExampleCommand_tag() {
	err := get.Command{Git: &kptfilev1alpha2.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "refs/tags/v1.0",
	}}.Run(fake.CtxWithEmptyPrinter())
	if err != nil {
		// handle error
	}
}

func ExampleCommand_commit() {
	err := get.Command{Git: &kptfilev1alpha2.Git{
		Repo: "https://github.com/example-org/example-repo",
		Ref:  "8186bef8e5c0621bf80fa8106bd595aae8b62884",
	}}.Run(fake.CtxWithEmptyPrinter())
	if err != nil {
		// handle error
	}
}

func ExampleCommand_subdir() {
	err := get.Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      "https://github.com/example-org/example-repo",
			Ref:       "v1.0",
			Directory: filepath.Join("path", "to", "package"),
		},
	}.Run(fake.CtxWithEmptyPrinter())
	if err != nil {
		// handle error
	}
}

func ExampleCommand_destination() {
	err := get.Command{
		Git: &kptfilev1alpha2.Git{
			Repo: "https://github.com/example-org/example-repo",
			Ref:  "v1.0",
		},
		Destination: "destination-dir"}.Run(fake.CtxWithEmptyPrinter())
	if err != nil {
		// handle error
	}
}
