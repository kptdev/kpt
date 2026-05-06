// Copyright 2025 The kpt Authors
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

package merge3

import (
	"path/filepath"
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/stretchr/testify/suite"
)

const (
	testDataDir = "testdata"
	originalPrr = "original.yaml"
	updatedPrr  = "updated.yaml"
	destPrr     = "destination.yaml"

	testAppName = "test-app"
	testCrName  = "test-fruit-store"
)

type Merge3TestSuite struct {
	suite.Suite
}

func TestMerge3(t *testing.T) {
	suite.Run(t, &Merge3TestSuite{})
}

type testCase struct {
	dir        string
	crds       []string
	checkFn    func(*Merge3TestSuite, fn.KubeObjects)
	skipReason string
}

func (t *Merge3TestSuite) commonTest(name string, tc testCase) {
	if tc.skipReason != "" {
		t.Run(name, func() { t.T().Skipf("skipping test %q: %s", name, tc.skipReason) })
		return
	}
	fullpath, err := filepath.Abs(filepath.Join(testDataDir, tc.dir))
	t.Require().NoError(err)
	orig := filepath.Join(fullpath, originalPrr)
	updated := filepath.Join(fullpath, updatedPrr)
	local := filepath.Join(fullpath, destPrr)

	var addSchemas []byte
	if len(tc.crds) > 0 {
		addSchemas = t.parseCrds(fullpath, tc.crds)
	}
	t.Run(name, func() { t.innerTest(orig, updated, local, addSchemas, tc.checkFn) })
}

func (t *Merge3TestSuite) TestBasic() {
	testCases := map[string]testCase{
		"simple-conflict": {
			dir:     "simple-conflict",
			checkFn: basicImageCheck,
		},
		// TODO: expand
		"simple-subpackage-conflict": {
			dir:     "simple-subpackage-conflict",
			checkFn: basicImageCheck,
		},
	}

	for name, tc := range testCases {
		t.commonTest(name, tc)
	}
}

func (t *Merge3TestSuite) TestOneKeyCrd() {
	testCases := map[string]testCase{
		"one-key-crd": {
			dir:     "one-key-crd",
			crds:    []string{"fruitstore.crd.yaml"},
			checkFn: assocListMergeCheck,
		},
		"one-key-crd-empty-orig": {
			dir:  "one-key-crd-empty-orig",
			crds: []string{"fruitstore.crd.yaml"},
			checkFn: makeFruitCheckFunc(10, map[string]int{
				"apple":  20,
				"grape":  5,
				"pear":   30,
				"banana": 3,
			}),
		},
		"one-key-crd-empty-updated": {
			dir:  "one-key-crd-empty-updated",
			crds: []string{"fruitstore.crd.yaml"},
			checkFn: makeFruitCheckFunc(10, map[string]int{
				"banana": 3,
			}),
		},
		"one-key-crd-empty-dest": {
			skipReason: "kyaml doesn't add the !!str tag to apple for some reason",
			dir:        "one-key-crd-empty-dest",
			crds:       []string{"fruitstore.crd.yaml"},
			checkFn: makeFruitCheckFunc(10, map[string]int{
				"apple": 20,
				"pear":  30,
			}),
		},
	}

	for name, tc := range testCases {
		t.commonTest(name, tc)
	}
}

func (t *Merge3TestSuite) TestInferAssocList() {
	t.T().Skipf("infer has been disabled")
	testCases := map[string]testCase{
		"infer-crd": {
			dir:     "infer-crd",
			checkFn: assocListMergeCheck,
		},
		"infer-crd-empty-orig": {
			dir: "infer-crd-empty-orig",
			checkFn: makeFruitCheckFunc(10, map[string]int{
				"apple":  20,
				"grape":  5,
				"pear":   30,
				"banana": 3,
			}),
		},
		"infer-crd-empty-updated": {
			dir: "infer-crd-empty-updated",
			checkFn: makeFruitCheckFunc(10, map[string]int{
				"banana": 3,
			}),
		},
		"infer-crd-empty-dest": {
			dir: "infer-crd-empty-dest",
			checkFn: makeFruitCheckFunc(10, map[string]int{
				"apple": 20,
				"pear":  30,
			}),
		},
	}

	for name, tc := range testCases {
		t.commonTest(name, tc)
	}
}
