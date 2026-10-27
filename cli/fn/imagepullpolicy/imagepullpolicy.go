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

// Package imagepullpolicy adapts the API image pull policy for pflag.
package imagepullpolicy

import (
	"fmt"
	"strings"

	"github.com/kptdev/kpt/api/runneroptions"
	"github.com/spf13/pflag"
)

// Flag adapts api ImagePullPolicy for pflag.
type Flag struct {
	value *runneroptions.ImagePullPolicy
}

// NewFlag returns a pflag.Value that writes into value.
func NewFlag(value *runneroptions.ImagePullPolicy) *Flag {
	return &Flag{value: value}
}

var _ pflag.Value = (*Flag)(nil)

var allImagePullPolicy = []runneroptions.ImagePullPolicy{
	runneroptions.AlwaysPull,
	runneroptions.IfNotPresentPull,
	runneroptions.NeverPull,
}

// String implements pflag.Value.
func (f *Flag) String() string {
	return string(*f.value)
}

// Set implements pflag.Value.
func (f *Flag) Set(v string) error {
	l := strings.ToLower(v)
	for _, c := range allImagePullPolicy {
		if strings.ToLower(string(c)) == l {
			*f.value = c
			return nil
		}
	}
	return fmt.Errorf("must be one of: %s", strings.Join(f.AllStrings(), ", "))
}

// Type implements pflag.Value.
func (f *Flag) Type() string {
	return "ImagePullPolicy"
}

// AllStrings returns the allowed image pull policy values.
func (f *Flag) AllStrings() []string {
	allStrings := make([]string, 0, len(allImagePullPolicy))
	for _, c := range allImagePullPolicy {
		allStrings = append(allStrings, string(c))
	}
	return allStrings
}

// HelpAllowedValues builds help text for the allowed values.
func (f *Flag) HelpAllowedValues() string {
	return "(one of " + strings.Join(f.AllStrings(), ", ") + ")"
}
