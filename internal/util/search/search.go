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

	// PutValue is the value to be put at to field
	// filtered by path and/or value
	PutValue string

	// PutComment is the comment to be added at to field
	PutComment string

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

	if sr.shouldPutValueByPath() {
		return object, sr.putValue(object)
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

		if sr.PutComment != "" {
			var err error
			node.LineComment, err = resolvePattern(node.Value, sr.ByValueRegex, sr.PutComment)
			if err != nil {
				return err
			}
		}

		if sr.PutValue != "" {
			// TODO: pmarupaka Check if the new value honors the openAPI schema and/or
			// current field type, throw error if it doesn't
			var err error
			node.Value, err = resolvePattern(node.Value, sr.ByValueRegex, sr.PutValue)
			if err != nil {
				return err
			}
			// When encoding, if this tag is unset the value type will be
			// implied from the node properties
			node.Tag = yaml.NodeTagEmpty
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

// putLiteral puts the value in the user specified sr.ByPath
func (sr *SearchReplace) putValue(object *yaml.RNode) error {
	path := strings.Split(sr.ByPath, PathDelimiter)
	// lookup(or create) node for n-1 path elements
	node, err := object.Pipe(yaml.LookupCreate(yaml.MappingNode, path[:len(path)-1]...))
	if err != nil {
		return errors.Wrap(err)
	}
	// set the last path element key with the input value
	sn := yaml.NewScalarRNode(sr.PutValue)
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
		Value:     sr.PutValue,
	}
	sr.Result = append(sr.Result, res)
	sr.Count++
	return nil
}

// shouldPutValueByPath returns true if only absolute path and literal are provided,
// so that the value can be directly put without needing to traverse the entire node,
// handles the case of adding non-existent field-value to node
func (sr *SearchReplace) shouldPutValueByPath() bool {
	return isAbsPath(sr.ByPath) &&
		!strings.Contains(sr.ByPath, "[") && // TODO: pmarupaka Support appending literal for arrays
		sr.ByValue == "" &&
		sr.ByValueRegex == "" &&
		sr.PutValue != ""
}

// resolvePattern takes the field value of a node, valueRegex provided by
// user from --by-value-regex, patternRegex provided by user from --put-value/--put-comment,
// and makes best effort to derive the corresponding capture groups and resolve the pattern
// refer to tests for expected behavior
func resolvePattern(fieldValue, valueRegex, patternRegex string) (string, error) {
	if valueRegex == "" {
		return patternRegex, nil
	}
	r, err := regexp.Compile(valueRegex)
	if err != nil {
		return "", errors.Errorf("failed to compile input pattern %q: %s", valueRegex, err.Error())
	}
	captureGroup := r.FindStringSubmatch(fieldValue)
	res := patternRegex
	for i, val := range captureGroup {
		if i == 0 {
			continue
		}
		res = strings.ReplaceAll(res, fmt.Sprintf("${%d}", i), val)
	}

	// make sure that all capture groups are resolved and throw error if they are not
	re := regexp.MustCompile(`\$\{([0-9]+)\}`)
	if re.Match([]byte(res)) {
		return "", errors.Errorf("unable to resolve capture groups")
	}

	return res, nil
}
