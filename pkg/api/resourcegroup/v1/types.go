// Copyright 2021,2026 The kpt Authors
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

// Package v1 defines ResourceGroup schema.
// Version: v1 (stable)
// swagger:meta
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	RGFileName = "resourcegroup.yaml"
	// RGInventoryIDLabel is the label name used for storing an inventory ID.
	RGInventoryIDLabel = common.InventoryLabel

	// Deprecated: prefer ResourceGroupGVK
	RGFileKind = "ResourceGroup"
	// Deprecated: prefer ResourceGroupGVK
	RGFileGroup = "kpt.dev"
	// Deprecated: prefer ResourceGroupGVK
	RGFileVersion = "v1"
	// Deprecated: prefer ResourceGroupGVK
	RGFileAPIVersion = RGFileGroup + "/" + RGFileVersion
)

// ResourceGroupGVK is the GroupVersionKind of ResourceGroup objects
func ResourceGroupGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "kpt.dev",
		Version: "v1",
		Kind:    "ResourceGroup",
	}
}

// DefaultMeta is the ResourceMeta for ResourceGroup instances.
var DefaultMeta = yaml.ResourceMeta{
	TypeMeta: yaml.TypeMeta{
		APIVersion: RGFileAPIVersion,
		Kind:       RGFileKind,
	},
}

// ResourceGroup contains the inventory information about a package managed with kpt.
// This is the stable v1 API for ResourceGroup.
// swagger:model resourcegroup
type ResourceGroup struct {
	yaml.ResourceMeta `yaml:",inline" json:",inline"`
}
