// Copyright 2019 Google LLC
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

package kio_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/kio"
)

func TestPipe(t *testing.T) {
	p := Pipeline{
		Inputs:  []Reader{},
		Filters: []Filter{},
		Outputs: []Writer{},
	}

	err := p.Execute()
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
}

func TestSlice_Write(t *testing.T) {

}
