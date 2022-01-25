// Copyright 2022 Google LLC
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

package kpt

import (
	"context"
	"fmt"
	"strings"
)

// PkgGetOpts are options for invoking kpt PkgGet
type PkgGetOpts struct {
	Strategy string
}

// PkgGet is a wrapper around `kpt pkg get`, running it on the package.
func PkgGet(ctx context.Context, packageURI string, version string, localDestDirectory string, opts PkgGetOpts) error {
	// https://kpt.dev/book/03-packages/01-getting-a-package
	// https://kpt.dev/reference/cli/pkg/get/
	args := []string{"pkg", "get"}
	if opts.Strategy != "" {
		args = append(args, "--strategy="+opts.Strategy)
	}
	versionSuffix := ""
	if version == "" {
		versionSuffix = "@main"
	} else {
		versionSuffix = "@" + version
	}
	args = append(args, packageURI+versionSuffix)
	args = append(args, localDestDirectory)

	workdir := "" // we expect localDestDirectory not to exist, so we can't switch into it
	_, _, err := execKpt(ctx, workdir, args, execKptOptions{})
	if err != nil {
		return fmt.Errorf("kpt pkg update (%s) failed: %w", strings.Join(args, " "), err)
	}

	return nil
}
