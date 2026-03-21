// Copyright 2026 The kpt and Nephio Authors
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

import "github.com/kptdev/kpt/pkg/lib/runneroptions"

// CELEvaluator is an alias for runneroptions.CELEnvironment so that runner.go
// can reference it within the fnruntime package without an import cycle.
type CELEvaluator = runneroptions.CELEnvironment
