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

// These tests depend on `diff` which is not available on Windows
//go:build !windows

// Package diff_test tests the diff package
package diff_test

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestCommand_Diff(t *testing.T) {
	testCases := map[string]struct {
		reposChanges              map[string][]testutil.Content
		updatedLocal              testutil.Content
		fetchRef                  string
		diffRef                   string
		diffType                  Type
		diffTool                  string
		diffOpts                  string
		expDiff                   string
		hasLocalSubpackageChanges bool
	}{

		// 1. add data to the upstream master branch
		// 2. commit and tag the upstream master branch
		// 3. add more data to the upstream master branch, commit it
		// 4. create a local clone at the tag
		// 5. add more data to the upstream master branch, commit it
		// 6. Run remote diff between upstream and the local fork.
		"remoteDiff": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Data:   testutil.Dataset2,
						Branch: "master",
						Tag:    "v2",
					},
					{
						Data: testutil.Dataset3,
					},
				},
			},
			fetchRef: "v2",
			diffRef:  "master",
			diffType: TypeRemote,
			diffTool: "diff",
			diffOpts: "-r -i -w",
			expDiff: `
41c41
<             - containerPort: 80
---
>             - containerPort: 8081
27,29c27,29
<     - name: "80"
<       port: 80
<       targetPort: 80
---
>     - name: "8081"
>       port: 8081
>       targetPort: 8081
`,
		},

		// 1. add data to the upstream master branch
		// 2. commit and tag the upstream master branch
		// 3. add more data to the upstream master branch, commit it
		// 4. create a local clone at the tag
		// 5. add more data to the upstream master branch, commit it
		// 6. Run combined diff between upstream and the local fork
		"combinedDiff": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Data:   testutil.Dataset2,
						Branch: "master",
						Tag:    "v2",
					},
					{
						Data: testutil.Dataset3,
					},
				},
			},
			fetchRef: "v2",
			diffRef:  "master",
			diffType: TypeCombined,
			diffTool: "diff",
			diffOpts: "-r -i -w",
			expDiff: `
41c41
<             - containerPort: 80
---
>             - containerPort: 8081
27,29c27,29
<     - name: "80"
<       port: 80
<       targetPort: 80
---
>     - name: "8081"
>       port: 8081
>       targetPort: 8081
`,
		},

		// 1. add data to the upstream master branch
		// 2. commit and tag the upstream master branch
		// 3. add more data to the upstream master branch, commit it
		// 4. create a local clone at the tag
		// 5. add more data to the upstream master branch, commit it
		// 6. Update the local fork with dataset3
		// 7. Run remote diff and verify the output
		"localDiff": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Data:   testutil.Dataset2,
						Branch: "master",
						Tag:    "v2",
					},
				},
			},
			updatedLocal: testutil.Content{
				Data: testutil.Dataset3,
			},
			fetchRef: "v2",
			diffRef:  "master",
			diffType: TypeCombined,
			diffTool: "diff",
			diffOpts: "-r -i -w",
			expDiff: `
41c41
<             - containerPort: 8081
---
>             - containerPort: 80
27,29c27,29
<     - name: "8081"
<       port: 8081
<       targetPort: 8081
---
>     - name: "80"
>       port: 80
>       targetPort: 80
`,
		},
		"nested local packages updated in local": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", "main", "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", "main", 0),
					).
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("5", "spec", "replicas")).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.SecretResource).
							WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("2", "spec", "replicas")),
					),
			},
			fetchRef: "main",
			diffRef:  "main",
			diffType: TypeCombined,
			diffTool: "diff",
			diffOpts: "-r -i -w",
			expDiff: `
9c9
<   replicas: 5
---
>   replicas: 3
locally changed: foo
			`,
			hasLocalSubpackageChanges: true,
		},
		"nested remote packages updated in local": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.SecretResource),
						Branch: "master",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", "main", "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", "main", 0),
					).
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("5", "spec", "replicas")).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "master", "resource-merge").
									WithUpstreamLockRef("foo", "/", "master", 0),
							).
							WithResource(pkgbuilder.SecretResource).
							WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("2", "spec", "replicas")),
					),
			},
			fetchRef: "main",
			diffRef:  "main",
			diffType: TypeCombined,
			diffTool: "diff",
			diffOpts: "-r -i -w",
			expDiff: `
9c9
<   replicas: 5
---
>   replicas: 3
			`,
		},

		//nolint:gocritic
		// TODO(mortent): Diff functionality must be updated to handle nested packages.
		"nested remote package updated in upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource,
								pkgbuilder.SetFieldPath("5", "spec", "replicas")),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.SecretResource),
						Branch: "master",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", "main", "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", "main", 0),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "master", "resource-merge").
									WithUpstreamLockRef("foo", "/", "master", 0),
							).
							WithResource(pkgbuilder.SecretResource),
					),
			},
			fetchRef: "main",
			diffRef:  "main",
			diffType: TypeCombined,
			diffTool: "diff",
			diffOpts: "-r -i -w",
			expDiff: `
9c9
<   replicas: 3
---
>   replicas: 5
		`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T:            t,
				ReposChanges: tc.reposChanges,
				GetRef:       tc.fetchRef,
			}
			defer g.Clean()

			if tc.updatedLocal.Pkg != nil || len(tc.updatedLocal.Data) > 0 {
				g.LocalChanges = []testutil.Content{
					tc.updatedLocal,
				}
			}
			if !g.Init() {
				return
			}

			diffOutput := &bytes.Buffer{}
			err := (&Command{
				Path:         g.LocalWorkspace.FullPackagePath(),
				Ref:          tc.diffRef,
				DiffType:     tc.diffType,
				DiffTool:     tc.diffTool,
				DiffToolOpts: tc.diffOpts,
				Output:       diffOutput,
			}).Run(fake.CtxWithDefaultPrinter())
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			filteredOutput := filterDiffMetadata(diffOutput)
			if tc.hasLocalSubpackageChanges {
				filteredOutput = regexp.MustCompile("Only in /(tmp|var).+:").ReplaceAllString(filteredOutput, "locally changed:")
			}
			assert.Equal(t, strings.TrimSpace(tc.expDiff)+"\n", filteredOutput)
		})
	}
}

func TestCommand_InvalidRef(t *testing.T) {
	reposChanges := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Data:   testutil.Dataset2,
				Branch: "master",
				Tag:    "v2",
			},
			{
				Data: testutil.Dataset3,
			},
		},
	}

	g := &testutil.TestSetupManager{
		T:            t,
		ReposChanges: reposChanges,
		GetRef:       "v2",
	}
	defer g.Clean()

	if !g.Init() {
		return
	}

	diffOutput := &bytes.Buffer{}
	err := (&Command{
		Path:         g.LocalWorkspace.FullPackagePath(),
		Ref:          "hurdygurdy", // ref should not exist in upstream
		DiffType:     TypeCombined,
		DiffTool:     "diff",
		DiffToolOpts: "-r -i -w",
		Output:       diffOutput,
	}).Run(fake.CtxWithDefaultPrinter())
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "unknown revision or path not in the working tree.")
}

// Validate that all three directories are staged and provided to diff command
func TestCommand_Diff3Parameters(t *testing.T) {
	reposChanges := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Data:   testutil.Dataset2,
				Branch: "master",
				Tag:    "v2",
			},
			{
				Data: testutil.Dataset3,
			},
		},
	}

	g := &testutil.TestSetupManager{
		T:            t,
		ReposChanges: reposChanges,
		GetRef:       "v2",
	}
	defer g.Clean()

	if !g.Init() {
		return
	}

	diffOutput := &bytes.Buffer{}
	err := (&Command{
		Path:         g.LocalWorkspace.FullPackagePath(),
		Ref:          "master",
		DiffType:     Type3Way,
		DiffTool:     "echo", // this is a proxy for 3 way diffing to validate we pass proper values
		DiffToolOpts: "",
		Output:       diffOutput,
	}).Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// Expect 3 value to be printed (1 per source)
	results := strings.Split(diffOutput.String(), " ")
	assert.Equal(t, 3, len(results))
	// Validate diff argument ordering
	assert.Contains(t, results[0], LocalPackageSource)
	assert.Contains(t, results[1], RemotePackageSource)
	assert.Contains(t, results[2], TargetRemotePackageSource)
}

// Tests against directories in different states
func TestCommand_NotAKptDirectory(t *testing.T) {
	// Initial test setup
	dir := t.TempDir()

	testCases := map[string]struct {
		directory string
	}{
		"Directory Is Not Kpt Package": {directory: dir},
		"Directory Does Not Exist":     {directory: "/not/a/directory"},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			diffOutput := &bytes.Buffer{}
			cmdErr := (&Command{
				Path:         tc.directory,
				Ref:          "master",
				DiffType:     TypeCombined,
				DiffTool:     "diff",
				DiffToolOpts: "-r -i -w",
				Output:       diffOutput,
			}).Run(fake.CtxWithDefaultPrinter())
			assert.Error(t, cmdErr)

			assert.Contains(t, cmdErr.Error(), "no such file or directory")
		})
	}
}

// filterDiffMetadata removes information from the diff output that is test-run
// specific for ex. removing directory name being used.
func filterDiffMetadata(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	b := &bytes.Buffer{}

	for scanner.Scan() {
		text := scanner.Text()
		// filter out the diff command that contains directory names
		if strings.HasPrefix(text, "diff ") {
			continue
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	return b.String()
}

func TestStagingDirectoryNames(t *testing.T) {
	var tests = []struct {
		source   string
		branch   string
		expected string
	}{
		{"source", "branch", "source-branch"},
		{"source", "refs/tags/version", "source-version"},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.expected, func(t *testing.T) {
			result := NameStagingDirectory(tt.source, tt.branch)
			assert.Equal(t, tt.expected, result)
		})
	}
}
