// Copyright 2022-2026 The kpt Authors
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

package kubeobject

import (
	"fmt"
	"sort"
	"strings"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// re-export for compatibility
type (
	Result   = fnresultv1.ResultItem
	Field    = fnresultv1.Field
	File     = framework.File
	Severity = framework.Severity
)

// re-export for compatibility
const (
	Error   = framework.Error
	Warning = framework.Warning
	Info    = framework.Info
)

type Results []*Result

// Errorf writes an Error level `result` to the results slice. It accepts arguments according to a format specifier.
// e.g.
// results.Errorf("bad kind %v", "invalid")
func (r *Results) Errorf(format string, a ...any) {
	errResult := &Result{Severity: Error, Message: fmt.Sprintf(format, a...)}
	*r = append(*r, errResult)
}

// ErrorE writes the `error` as an Error level `result` to the results slice.
// e.g.
//
//	err := error.New("test)
//	results.ErrorE(err)
func (r *Results) ErrorE(err error) {
	errResult := &Result{Severity: Error, Message: err.Error()}
	*r = append(*r, errResult)
}

// Infof writes an Info level `result` to the results slice. It accepts arguments according to a format specifier.
// e.g.
//
//	results.Infof("update %v %q ", "ConfigMap", "kptfile.kpt.dev")
func (r *Results) Infof(format string, a ...any) {
	infoResult := &Result{Severity: Info, Message: fmt.Sprintf(format, a...)}
	*r = append(*r, infoResult)
}

// Warningf writes a Warning level `result` to the results slice. It accepts arguments according to a format specifier.
// e.g.
//
//	results.Warningf("bad kind %q", "invalid")
func (r *Results) Warningf(format string, a ...any) {
	warnResult := &Result{Severity: Warning, Message: fmt.Sprintf(format, a...)}
	*r = append(*r, warnResult)
}

// WarningE writes an error as a Warning level `result` to the results slice.
// Normally this function can be used for cases that need error tolerance.
func (r *Results) WarningE(err error) {
	warnResult := &Result{Severity: Warning, Message: err.Error()}
	*r = append(*r, warnResult)
}

func (r *Results) String() string {
	var results []string
	for _, result := range *r {
		results = append(results, result.String())
	}
	return strings.Join(results, "\n---\n")
}

// Error enables Results to be returned as an error
func (r Results) Error() string {
	var msgs []string
	for _, i := range r {
		msgs = append(msgs, i.String())
	}
	return strings.Join(msgs, "\n\n")
}

// ExitCode provides the exit code based on the result's severity
func (r Results) ExitCode() int {
	for _, i := range r {
		if i.Severity == Error {
			return 1
		}
	}
	return 0
}

// Sort performs an in place stable sort of Results
func (r Results) Sort() {
	sort.SliceStable(r, func(i, j int) bool {
		if fileLess(r, i, j) != 0 {
			return fileLess(r, i, j) < 0
		}
		if severityLess(r, i, j) != 0 {
			return severityLess(r, i, j) < 0
		}
		return resultToString(*r[i]) < resultToString(*r[j])
	})
}

func severityLess(items Results, i, j int) int {
	severityToNumber := map[Severity]int{
		Error:   0,
		Warning: 1,
		Info:    2,
	}

	severityLevelI, found := severityToNumber[items[i].Severity]
	if !found {
		severityLevelI = 3
	}
	severityLevelJ, found := severityToNumber[items[j].Severity]
	if !found {
		severityLevelJ = 3
	}
	return severityLevelI - severityLevelJ
}

func fileLess(items Results, i, j int) int {
	var fileI, fileJ File
	if items[i].File == nil {
		fileI = File{}
	} else {
		fileI = *items[i].File
	}
	if items[j].File == nil {
		fileJ = File{}
	} else {
		fileJ = *items[j].File
	}
	if fileI.Path != fileJ.Path {
		if fileI.Path < fileJ.Path {
			return -1
		}
		return 1
	}
	return fileI.Index - fileJ.Index
}

func resultToString(item Result) string {
	return fmt.Sprintf("resource-ref:%s,field:%s,message:%s",
		item.ResourceRef, item.Field, item.Message)
}

func ErrorConfigFileResult(err error, path string) *Result {
	return ConfigFileResult(err.Error(), path, Error)
}

func ConfigFileResult(msg, path string, severity Severity) *Result {
	return &Result{
		Message:  msg,
		Severity: severity,
		File: &File{
			Path: path,
		},
	}
}

func ErrorResult(err error) *Result {
	return GeneralResult(err.Error(), Error)
}

func GeneralResult(msg string, severity Severity) *Result {
	return &Result{
		Message:  msg,
		Severity: severity,
	}
}

func ErrorConfigObjectResult(err error, obj *KubeObject) *Result {
	return ConfigObjectResult(err.Error(), obj, Error)
}

func ConfigObjectResult(msg string, obj *KubeObject, severity Severity) *Result {
	return &Result{
		Message:  msg,
		Severity: severity,
		ResourceRef: &yaml.ResourceIdentifier{
			TypeMeta: yaml.TypeMeta{
				APIVersion: obj.GetAPIVersion(),
				Kind:       obj.GetKind(),
			},
			NameMeta: yaml.NameMeta{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		},
		File: &File{
			Path:  obj.PathAnnotation(),
			Index: obj.IndexAnnotation(),
		},
	}
}
