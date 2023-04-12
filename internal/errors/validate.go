// Copyright 2021 The kpt Authors
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

package errors

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/util/strings"
)

// ValidationError is an error type used when validation fails.
type ValidationError struct {
	Violations Violations
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for fields %s",
		strings.JoinStringsWithQuotes(e.Violations.Fields()))
}

type ViolationType string

const (
	Missing ViolationType = "missing"
	Invalid ViolationType = "invalid"
)

type Violations []Violation

func (v Violations) Fields() []string {
	var fields []string
	for _, v := range v {
		fields = append(fields, v.Field)
	}
	return fields
}

type Violation struct {
	Field  string
	Value  string
	Type   ViolationType
	Reason string
}
