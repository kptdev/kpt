// Copyright 2022 The kpt Authors
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

package builtins

import (
	"bufio"
	"os"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/builtins/builtintypes"
	"github.com/stretchr/testify/assert"
)

func TestGetBuiltinFunction(t *testing.T) {
	assert.PanicsWithError(t, "unsupported builtin function configuration <nil> of type <nil>", func() { GetBuiltinFn(nil) })
	assert.PanicsWithError(t, "unsupported builtin function configuration hello of type string", func() { GetBuiltinFn("hello") })

	packageContextGenerator := GetBuiltinFn(&builtintypes.PackageConfig{})
	assert.NotNil(t, packageContextGenerator)

	fileHandle, err := os.Open("/")
	assert.Nil(t, err)
	err = packageContextGenerator.Run(bufio.NewReader(fileHandle), nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "directory")
}
