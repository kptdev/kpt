// Copyright 2021 Google LLC
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

// Package defines ResourceGroup schema.
// Version: v1alpha1
// swagger:meta
package v1alpha1

import (
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	RGFileName       = "resourcegroup.yaml"
	RGFileKind       = "ResourceGroup"
	RGFileGroup      = "kpt.dev"
	RGFileVersion    = "v1alpha1"
	RGFileAPIVersion = RGFileGroup + "/" + RGFileVersion
	// RGInventoryIDLabel is the label name used for storing an inventory ID.
	RGInventoryIDLabel = common.InventoryLabel
)

// DefaultMeta is the ResourceMeta for ResourceGroup instances.
var DefaultMeta = yaml.ResourceMeta{
	TypeMeta: yaml.TypeMeta{
		APIVersion: RGFileAPIVersion,
		Kind:       RGFileKind,
	},
}

// ResourceGroup contains the inventory information about a package managed with kpt.
// swagger:model resourcegroup
type ResourceGroup struct {
	yaml.ResourceMeta `yaml:",inline" json:",inline"`
}
