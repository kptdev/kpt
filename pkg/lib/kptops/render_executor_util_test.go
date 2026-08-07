// Copyright 2026 The kpt Authors
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

package kptops

import (
	"slices"
	"strings"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/stretchr/testify/require"
)

func TestResolveFullDisplayName(t *testing.T) {
	testCases := map[string]struct {
		opts     runneroptions.LogOptions
		expected string
	}{
		"old kpt format": {
			opts: runneroptions.LogOptions{
				PkgNameFormat: "%s",
				PkgNameSep:    "/",
				PkgNameID:     runneroptions.DirName,
			},
			expected: "root/subpkg/child",
		},
		"new kpt format": {
			opts: runneroptions.LogOptions{
				PkgNameFormat: "%s",
				PkgNameSep:    "/",
				PkgNameID:     runneroptions.KptfileMeta,
			},
			expected: "root-package/sub-package/child-package",
		},
		"Porch format": {
			opts: runneroptions.LogOptions{
				PkgNameFormat: "repo.%s.v1",
				PkgNameSep:    ".",
				PkgNameID:     runneroptions.DirName,
			},
			expected: "repo.root.subpkg.child.v1",
		},
	}

	r, _, _ := setupRendererTest(t, false)

	rootpkg, err := pkg.New(r.FileSystem, "/root")
	require.NoError(t, err)

	rootsubpkgs, err := rootpkg.DirectSubpackages()
	require.NoError(t, err)
	require.Len(t, rootsubpkgs, 2)

	subidx := slices.IndexFunc(rootsubpkgs, func(p *pkg.Pkg) bool {
		return strings.Contains(string(p.UniquePath), "subpkg")
	})

	require.NotEqual(t, -1, subidx, "Could not find \"subpkg\"")

	child1pkg := rootsubpkgs[subidx]

	child1subpkgs, err := child1pkg.DirectSubpackages()
	require.NoError(t, err)
	require.Len(t, child1subpkgs, 1)

	child2pkg := child1subpkgs[0]

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actual := resolveFullDisplayName(child2pkg, tc.opts)
			require.Equal(t, tc.expected, actual)
		})
	}
}
