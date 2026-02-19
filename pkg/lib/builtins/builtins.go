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

// Package builtins passes builtin functions back to the caller for recognized configuration types
package builtins

import (
	"fmt"
	"reflect"

	"github.com/kptdev/kpt/internal/builtins"
	builtintypes "github.com/kptdev/kpt/pkg/lib/builtins/builtintypes"
)

func GetBuiltinFn(config any) builtintypes.BuiltinFunction {
	if reflect.TypeOf(config) == reflect.TypeFor[*builtintypes.PackageConfig]() {
		packageConfig := config.(*builtintypes.PackageConfig)
		return &builtins.PackageContextGenerator{
			PackageConfig: packageConfig,
		}
	}
	panic(fmt.Errorf("unsupported builtin function configuration %v of type %v", config, reflect.TypeOf(config)))
}
