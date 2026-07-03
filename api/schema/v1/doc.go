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

// Package v1 provides types for identifying Kubernetes API resources by
// group, version, and kind without depending on k8s.io/apimachinery.
//
// Portions of this package are adapted from the Kubernetes apimachinery project:
// https://github.com/kubernetes/apimachinery/blob/v0.34.9/pkg/runtime/schema/group_version.go
//
// Copyright 2015 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0
package v1
