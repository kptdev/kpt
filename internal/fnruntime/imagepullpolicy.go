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

package fnruntime

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

type ImagePullPolicy string

const (
	AlwaysPull       ImagePullPolicy = "Always"
	IfNotPresentPull ImagePullPolicy = "IfNotPresent"
	NeverPull        ImagePullPolicy = "Never"
)

var allImagePullPolicy = []ImagePullPolicy{
	AlwaysPull,
	IfNotPresentPull,
	NeverPull,
}

// ImagePullPolicy can be used in pflag
var _ pflag.Value = ((*ImagePullPolicy)(nil))

// String implements pflag.Value and fmt.Stringer
func (e *ImagePullPolicy) String() string {
	return string(*e)
}

// Set implements pflag.Value
func (e *ImagePullPolicy) Set(v string) error {
	l := strings.ToLower(v)
	for _, c := range allImagePullPolicy {
		s := strings.ToLower(string(c))
		if s == l {
			*e = c
			return nil
		}
	}
	return fmt.Errorf("must must be one of " + strings.Join(e.AllStrings(), ", "))
}

func (e *ImagePullPolicy) AllStrings() []string {
	var allStrings []string
	for _, c := range allImagePullPolicy {
		allStrings = append(allStrings, string(c))
	}
	return allStrings
}

// HelpAllowedValues builds help text for the allowed values
func (e *ImagePullPolicy) HelpAllowedValues() string {
	return "(one of " + strings.Join(e.AllStrings(), ", ") + ")"
}

// Type implements pflag.Value
func (e *ImagePullPolicy) Type() string {
	return "ImagePullPolicy"
}
