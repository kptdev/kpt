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

// Package kptops's render-specific wrapper around pkg/lib/krmvalidate's
// shared non-KRM pre-scan.
package kptops

import (
	"github.com/kptdev/kpt/pkg/lib/krmvalidate"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// AutoIgnoreNonKRMAnnotation re-exports krmvalidate.AutoIgnoreNonKRMAnnotation.
const AutoIgnoreNonKRMAnnotation = krmvalidate.AutoIgnoreNonKRMAnnotation

// NonKRMPreScanResult re-exports krmvalidate.ScanResult.
type NonKRMPreScanResult = krmvalidate.ScanResult

// PreScanNonKRM runs krmvalidate.Scan for pkgPath.
func PreScanNonKRM(fsys filesys.FileSystem, pkgPath string, printf func(format string, args ...any)) (*NonKRMPreScanResult, error) {
	return krmvalidate.Scan(fsys, pkgPath, printf)
}
