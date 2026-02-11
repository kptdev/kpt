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

// Package builtintypes holds types for kpt file package context
package builtintypes

import (
	"io"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

const (
	PkgContextFile = "package-context.yaml"
	PkgContextName = "kptfile.kpt.dev"

	ConfigKeyPackagePath = "package-path"
)

// BuiltinFunction returns a reference to a builtin function
type BuiltinFunction interface {
	Run(io.Reader, io.Writer) error
	Process(*framework.ResourceList) error
}

// PackageConfig holds package automatic configuration
type PackageConfig struct {
	// PackagePath is the path to the package, as determined by the names of the parent packages.
	// The path to a package is the parent package path joined with the package name.
	PackagePath string
}
