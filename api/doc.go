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

// Package apis contains the kpt API type definitions.
package apis

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0 object:headerFile="../hack/boilerplate.go.txt",year=$YEAR_GEN applyconfiguration:headerFile="../hack/boilerplate.go.txt" paths=./...
// Drop generated helpers that pull in k8s.io/apimachinery; typed ACs are enough.
//go:generate rm -rf generated/applyconfigurations/utils.go generated/applyconfigurations/internal
