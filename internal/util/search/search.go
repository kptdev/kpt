// Copyright 2020 Google LLC
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

package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/setters2"
	"sigs.k8s.io/kustomize/kyaml/setters2/settersutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const PathDelimiter = "."

// SearchReplace struct holds the input parameters and results for
// Search and Replace operations on resource configs
type SearchReplace struct {
	// ByValue is the value of the field to be matched
	ByValue string

	// ByValueRegex is the value regex of the field to be matched
	ByValueRegex string

	regex *regexp.Regexp

	// ByPath is the path of the field to be matched
	ByPath string

	// Count is the number of matches
	Count int

	// PutLiteral is the literal to be put at to field
	// filtered by path and/or value
	PutLiteral string

	// PutPattern is the setters reference comment to be added at to field
	PutPattern string

	filePath string

	PackagePath string

	// Result stores the result of executing the command
	Result []SearchResult
}

type SearchResult struct {
	// file path of the matching field
	FilePath string

	// field path of the matching field
	FieldPath string

	// value of the matching field
	Value string
}

// Perform performs the search and replace operation on each node in the package path
func (sr *SearchReplace) Perform(resourcesPath string) error {
	inout := &kio.LocalPackageReadWriter{
		PackagePath:     resourcesPath,
		NoDeleteFiles:   true,
		PackageFileName: kptfile.KptFileName,
	}

	if sr.ByValueRegex != "" {
		re, err := regexp.Compile(sr.ByValueRegex)
		if err != nil {
			return errors.Wrap(err)
		}
		sr.regex = re
	}

	return kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{kio.FilterAll(sr)},
		Outputs: []kio.Writer{inout},
	}.Execute()
}

// Filter parses input node and performs search and replace operation on the node
func (sr *SearchReplace) Filter(object *yaml.RNode) (*yaml.RNode, error) {
	filePath, _, err := kioutil.GetFileAnnotations(object)
	if err != nil {
		return object, err
	}
	sr.filePath = filePath

	if sr.shouldPutLiteralByPath() {
		return object, sr.putLiteral(object)
	}

	// traverse the node to perform search/put operation
	err = accept(sr, object)
	return object, err
}

// visitMapping parses mapping node
func (sr *SearchReplace) visitMapping(object *yaml.RNode, path string) error {
	return nil
}

// visitSequence parses sequence node
func (sr *SearchReplace) visitSequence(object *yaml.RNode, path string) error {
	return nil
}

// visitScalar parses scalar node
func (sr *SearchReplace) visitScalar(object *yaml.RNode, path string) error {
	return sr.matchAndReplace(object.Document(), path)
}

func (sr *SearchReplace) matchAndReplace(node *yaml.Node, path string) error {
	pathMatch := sr.pathMatch(path)
	// check if the node value matches with the input by-value-regex or the by-value
	// empty node values are not matched
	valueMatch := (sr.ByValue != "" && sr.ByValue == node.Value) || sr.regexMatch(node.Value)

	if (valueMatch && pathMatch) || (valueMatch && sr.ByPath == "") ||
		(pathMatch && sr.ByValue == "" && sr.ByValueRegex == "") {
		sr.Count++

		if sr.PutLiteral != "" {
			// TODO: pmarupaka Check if the new value honors the openAPI schema and/or
			// current field type, throw error if it doesn't
			node.Value = sr.PutLiteral
			// When encoding, if this tag is unset the value type will be
			// implied from the node properties
			node.Tag = yaml.NodeTagEmpty
		}

		if sr.PutPattern != "" {
			var err error
			pattern := sr.PutPattern
			if sr.ByValueRegex != "" {
				pattern, err = setterPatternFromRegex(node.Value, sr.ByValueRegex, sr.PutPattern)
				if err != nil {
					return err
				}
			}
			node.LineComment = fmt.Sprintf(`kpt-set: %s`, pattern)
		}

		if sr.filePath != "" {
			nodeVal, err := yaml.String(node)
			if err != nil {
				return err
			}
			res := SearchResult{
				FilePath:  sr.filePath,
				FieldPath: strings.TrimPrefix(path, PathDelimiter),
				Value:     strings.TrimSpace(nodeVal),
			}
			sr.Result = append(sr.Result, res)
		}
	}
	return nil
}

// regexMatch checks if ValueRegex in SearchReplace struct matches with the input
// value, returns error if any
func (sr *SearchReplace) regexMatch(value string) bool {
	if sr.ByValueRegex == "" {
		return false
	}
	return sr.regex.Match([]byte(value))
}

// putLiteral puts the literal in the user specified sr.ByPath
func (sr *SearchReplace) putLiteral(object *yaml.RNode) error {
	path := strings.Split(sr.ByPath, PathDelimiter)
	// lookup(or create) node for n-1 path elements
	node, err := object.Pipe(yaml.LookupCreate(yaml.MappingNode, path[:len(path)-1]...))
	if err != nil {
		return errors.Wrap(err)
	}
	// set the last path element key with the input value
	sn := yaml.NewScalarRNode(sr.PutLiteral)
	// When encoding, if this tag is unset the value type will be
	// implied from the node properties
	sn.YNode().Tag = yaml.NodeTagEmpty
	err = node.PipeE(yaml.SetField(path[len(path)-1], sn))
	if err != nil {
		return errors.Wrap(err)
	}
	res := SearchResult{
		FilePath:  sr.filePath,
		FieldPath: sr.ByPath,
		Value:     sr.PutLiteral,
	}
	sr.Result = append(sr.Result, res)
	sr.Count++
	return nil
}

// shouldPutLiteralByPath returns true if only absolute path and literal are provided,
// so that the value can be directly put without needing to traverse the entire node,
// handles the case of adding non-existent field-value to node
func (sr *SearchReplace) shouldPutLiteralByPath() bool {
	return isAbsPath(sr.ByPath) &&
		!strings.Contains(sr.ByPath, "[") && // TODO: pmarupaka Support appending literal for arrays
		sr.ByValue == "" &&
		sr.ByValueRegex == "" &&
		sr.PutLiteral != ""
}

// setterPatternFromRegex takes the field value of a node, valueRegex provided by
// user from --by-value-regex, patternRegex provided by user from --put-pattern,
// and makes best effort to derive the corresponding setter pattern comment to be
// added as line comment to the node.
// e.g. fieldValue: my-project-foo, valueRegex: my-project-*, patternRegex: ${project-id}-*
// the output pattern will be ${project-id}-foo
// in case the valueRegex is vague and not enough to derive the setter values, it
// returns an error
func setterPatternFromRegex(fieldValue, valueRegex, patternRegex string) (string, error) {
	settersValues, err := settersValues(valueRegex, patternRegex)
	if err != nil {
		return "", err
	}

	pattern := fieldValue
	// derive the pattern from the field value by replacing setter values
	// with setter name markers
	// e.g. if field value is "my-project-foo", input PutPattern is "${project]-*", and
	// value for setter "project" is "my-project",
	// the pattern to be added as comment is ${project]-foo
	for sn, sv := range settersValues {
		pattern = strings.ReplaceAll(clean(pattern), clean(sv), clean(sn))
	}

	return pattern, nil
}

// settersValues returns the values for the setters present in patternRegex,
// returns error if the setter values can't be derived from the pattern
// e.g. valueRegex = my-project-*, patternRegex = ${project-id}-* returns
// map[project-id]my-project
// e.g. valueRegex = nginx-*, patternRegex = ${image}:${tag}-* returns error
// as setter values can't be derived from valueRegex
func settersValues(valueRegex, patternRegex string) (map[string]string, error) {
	// extract setter name tokens from pattern enclosed in ${}
	re := regexp.MustCompile(`\$\{([^}]*)\}`)
	markers := re.FindAllString(patternRegex, -1)
	if len(markers) == 0 {
		return nil, errors.Errorf("unable to find setter names in pattern, " +
			"setter names must be enclosed in ${}")
	}
	sc := settersutil.SubstitutionCreator{
		FieldValue: valueRegex,
		Pattern:    patternRegex,
	}
	for _, m := range markers {
		sc.Values = append(sc.Values, setters2.Value{
			Marker: m,
		})
	}
	res, err := sc.GetValuesForMarkers()
	if err != nil {
		return nil, errors.Errorf("unable to derive setter values from the provided pattern, " +
			"please ensure value-regex matches provided setter pattern")
	}
	return res, nil
}

// clean trims the string and removes quotes around it
func clean(input string) string {
	return strings.Trim(strings.TrimSpace(input), `"`)
}
