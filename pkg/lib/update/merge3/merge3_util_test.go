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
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

func (t *Merge3TestSuite) findWithName(slice []*fn.SubObject, name string) *fn.SubObject {
	t.T().Helper()
	for _, so := range slice {
		n, found, err := so.NestedString("name")
		if !found || err != nil {
			continue
		}
		if n == name {
			return so
		}
	}
	t.FailNow(fmt.Sprintf("no object with name %q was found in slice", name))
	return nil
}

func (t *Merge3TestSuite) getSlice(obj *fn.SubObject, path ...string) []*fn.SubObject {
	slice, found, err := obj.NestedSlice(path...)
	t.Require().NoError(err)
	t.Require().Truef(found, "%s should exist", strings.Join(path, "."))

	return slice
}

func (t *Merge3TestSuite) assertVal(obj *fn.SubObject, expected any, path ...string) {
	t.T().Helper()
	var (
		val   any
		found bool
		err   error
	)

	switch expected.(type) {
	case string:
		val, found, err = obj.NestedString(path...)
	case int:
		val, found, err = obj.NestedInt(path...)
	case bool:
		val, found, err = obj.NestedBool(path...)
	}

	t.Require().NoError(err)
	t.Require().Truef(found, "%s should exist", strings.Join(path, ""))
	t.Require().Equal(expected, val)
}

func (t *Merge3TestSuite) parseCrds(directory string, crds []string) []byte {
	definitions := map[string]spec.Schema{}
	for _, crd := range crds {
		parsed := t.parseCrd(directory, crd)
		maps.Copy(definitions, parsed)
	}

	addSchemas := map[string]map[string]spec.Schema{
		"definitions": definitions,
	}

	marshalled, err := yaml.Marshal(addSchemas)
	t.Require().NoError(err)

	return marshalled
}

func (t *Merge3TestSuite) parseCrd(directory string, crd string) map[string]spec.Schema {
	t.T().Helper()
	path := filepath.Join(directory, crd)
	rn, err := kyaml.ReadFile(path)
	t.Require().NoError(err)
	schemas, err := SchemasFromCrdRNode(rn)
	t.Require().NoError(err)
	return schemas
}

func (t *Merge3TestSuite) innerTest(original, updated, dest string, addSchemas []byte, checkFn func(*Merge3TestSuite, fn.KubeObjects)) {
	o, u, d := t.parsePrrsToKubeObjects(original, updated, dest)

	openapi.ResetOpenAPI()
	result, err := Merge(o, u, d, addSchemas)
	t.Require().NoError(err)

	checkFn(t, result)
}

func (t *Merge3TestSuite) parsePrrsToKubeObjects(original, updated, dest string) (orig, upd, dst fn.KubeObjects) {
	t.T().Helper()
	o, u, d := t.parsePrrs(original, updated, dest)
	oko, _, err := fn.ReadKubeObjectsFromPackage(o)
	t.Require().NoError(err)
	uko, _, err := fn.ReadKubeObjectsFromPackage(u)
	t.Require().NoError(err)
	dko, _, err := fn.ReadKubeObjectsFromPackage(d)
	t.Require().NoError(err)
	return oko, uko, dko
}

func (t *Merge3TestSuite) parsePrrs(original, updated, dest string) (orig, upd, dst map[string]string) {
	t.T().Helper()
	return t.parsePrr(original), t.parsePrr(updated), t.parsePrr(dest)
}

func (t *Merge3TestSuite) parsePrr(path string) map[string]string {
	t.T().Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.FailNow("could not read file", err)
	}
	prr := &MergeTestResources{}
	if err = yaml.Unmarshal(content, prr); err != nil {
		t.FailNow("could not unmarshal PackageRevisionResources", err)
	}
	return prr.Spec.Resources
}

func nameIs(name string) func(ko *fn.KubeObject) bool {
	return func(ko *fn.KubeObject) bool {
		return ko.GetName() == name
	}
}

func basicImageCheck(t *Merge3TestSuite, kos fn.KubeObjects) {
	deplObj, err := kos.Where(nameIs(testAppName)).EnsureSingleItem()
	t.Require().NoError(err)

	containers := t.getSlice(&deplObj.SubObject, "spec", "template", "spec", "containers")

	t.assertVal(containers[0], "test-image:updated", "image")
}

func assocListMergeCheck(t *Merge3TestSuite, kos fn.KubeObjects) {
	makeFruitCheckFunc(10, map[string]int{
		"apple":  20,
		"grape":  5,
		"banana": 3,
		"pear":   30,
	})(t, kos)
}

//nolint:unparam
func makeFruitCheckFunc(expectedTemp int, expectedFruits map[string]int) func(*Merge3TestSuite, fn.KubeObjects) {
	return func(t *Merge3TestSuite, kos fn.KubeObjects) {
		obj, err := kos.Where(nameIs(testCrName)).EnsureSingleItem()
		t.Require().NoError(err)

		t.assertVal(&obj.SubObject, expectedTemp, "spec", "preferredTemperature")

		fruits := t.getSlice(&obj.SubObject, "spec", "fruits")
		t.Require().Len(fruits, len(expectedFruits))

		for name, amount := range expectedFruits {
			so := t.findWithName(fruits, name)
			t.assertVal(so, amount, "amount")
		}
	}
}
