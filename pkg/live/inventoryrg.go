// Copyright 2020 Google LLC
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

package live

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceGroupGVK is the group/version/kind of the custom
// resource used to store inventory.
var ResourceGroupGVK = schema.GroupVersionKind{
	Group:   "kpt.dev",
	Version: "v1alpha1",
	Kind:    "ResourceGroup",
}

// InventoryResourceGroup wraps a ResourceGroup resource and implements
// the Inventory and InventoryInfo interface. This wrapper loads and stores the
// object metadata (inventory) to and from the wrapped ResourceGroup.
type InventoryResourceGroup struct {
	inv      *unstructured.Unstructured
	objMetas []object.ObjMetadata
}

var _ inventory.Inventory = &InventoryResourceGroup{}
var _ inventory.InventoryInfo = &InventoryResourceGroup{}

// WrapInventoryObj takes a passed ResourceGroup (as a resource.Info),
// wraps it with the InventoryResourceGroup and upcasts the wrapper as
// an the Inventory interface.
func WrapInventoryObj(obj *unstructured.Unstructured) inventory.Inventory {
	if obj != nil {
		klog.V(4).Infof("wrapping Inventory obj: %s/%s\n", obj.GetNamespace(), obj.GetName())
	}
	return &InventoryResourceGroup{inv: obj}
}

func WrapInventoryInfoObj(obj *unstructured.Unstructured) inventory.InventoryInfo {
	if obj != nil {
		klog.V(4).Infof("wrapping InventoryInfo obj: %s/%s\n", obj.GetNamespace(), obj.GetName())
	}
	return &InventoryResourceGroup{inv: obj}
}

func InvToUnstructuredFunc(inv inventory.InventoryInfo) *unstructured.Unstructured {
	switch invInfo := inv.(type) {
	case *InventoryResourceGroup:
		return invInfo.inv
	default:
		return nil
	}
}

// Name(), Namespace(), and ID() are InventoryResourceGroup functions to
// implement the InventoryInfo interface.
func (icm *InventoryResourceGroup) Name() string {
	return icm.inv.GetName()
}

func (icm *InventoryResourceGroup) Namespace() string {
	return icm.inv.GetNamespace()
}

func (icm *InventoryResourceGroup) ID() string {
	labels := icm.inv.GetLabels()
	if val, found := labels[common.InventoryLabel]; found {
		return val
	}
	return ""
}

// Load is an Inventory interface function returning the set of
// object metadata from the wrapped ResourceGroup, or an error.
func (icm *InventoryResourceGroup) Load() ([]object.ObjMetadata, error) {
	objs := []object.ObjMetadata{}
	if icm.inv == nil {
		return objs, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("loading inventory...")
	items, exists, err := unstructured.NestedSlice(icm.inv.Object, "spec", "resources")
	if err != nil {
		err := fmt.Errorf("error retrieving object metadata from inventory object")
		return objs, err
	}
	if !exists {
		klog.V(4).Infof("Inventory (spec.resources) does not exist")
		return objs, nil
	}
	klog.V(4).Infof("loading %d inventory items", len(items))
	for _, itemUncast := range items {
		item := itemUncast.(map[string]interface{})
		namespace, _, err := unstructured.NestedString(item, "namespace")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		name, _, err := unstructured.NestedString(item, "name")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		group, _, err := unstructured.NestedString(item, "group")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		kind, _, err := unstructured.NestedString(item, "kind")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		groupKind := schema.GroupKind{
			Group: strings.TrimSpace(group),
			Kind:  strings.TrimSpace(kind),
		}
		klog.V(4).Infof("creating obj metadata: %s/%s/%s", namespace, name, groupKind)
		objMeta, err := object.CreateObjMetadata(namespace, name, groupKind)
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		objs = append(objs, objMeta)
	}
	return objs, nil
}

// Store is an Inventory interface function implemented to store
// the object metadata in the wrapped ResourceGroup. Actual storing
// happens in "GetObject".
func (icm *InventoryResourceGroup) Store(objMetas []object.ObjMetadata) error {
	icm.objMetas = objMetas
	return nil
}

// GetObject returns the wrapped object (ResourceGroup) as a resource.Info
// or an error if one occurs.
func (icm *InventoryResourceGroup) GetObject() (*unstructured.Unstructured, error) {
	if icm.inv == nil {
		return nil, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("getting inventory resource group")
	// Create a slice of Resources as empty Interface
	klog.V(4).Infof("Creating list of %d resources", len(icm.objMetas))
	var objs []interface{}
	for _, objMeta := range icm.objMetas {
		klog.V(4).Infof("storing inventory obj refercence: %s/%s", objMeta.Namespace, objMeta.Name)
		objs = append(objs, map[string]interface{}{
			"group":     objMeta.GroupKind.Group,
			"kind":      objMeta.GroupKind.Kind,
			"namespace": objMeta.Namespace,
			"name":      objMeta.Name,
		})
	}
	// Create the inventory object by copying the template.
	invCopy := icm.inv.DeepCopy()
	// Adds or clears the inventory ObjMetadata to the ResourceGroup "spec.resources" section
	if len(objs) == 0 {
		klog.V(4).Infoln("clearing inventory resources")
		unstructured.RemoveNestedField(invCopy.UnstructuredContent(),
			"spec", "resources")
	} else {
		klog.V(4).Infof("storing inventory (%d) resources", len(objs))
		err := unstructured.SetNestedSlice(invCopy.UnstructuredContent(),
			objs, "spec", "resources")
		if err != nil {
			return nil, err
		}
	}
	return invCopy, nil
}

// IsResourceGroupInventory returns true if the passed object is
// a ResourceGroup inventory object; false otherwise. If an error
// occurs, then false is returned and the error.
func IsResourceGroupInventory(obj *unstructured.Unstructured) (bool, error) {
	if obj == nil {
		return false, fmt.Errorf("inventory object is nil")
	}
	if !inventory.IsInventoryObject(obj) {
		return false, nil
	}
	invGK := obj.GetObjectKind().GroupVersionKind().GroupKind()
	if ResourceGroupGVK.GroupKind() != invGK {
		return false, nil
	}
	return true, nil
}

// CustomResourceDefinition schema, without specific version. The default version
// is returned when the RESTMapper returns a RESTMapping for this GroupKind.
var crdGroupKind = schema.GroupKind{
	Group: "apiextensions.k8s.io",
	Kind:  "CustomResourceDefinition",
}

// ApplyResourceGroupCRD applies the custom resource definition for the
// ResourceGroup. The apiextensions version applied is based on the RESTMapping
// returned by the RESTMapper. Returns an error if one occurs, including an
// "Already Exists" error.
func ApplyResourceGroupCRD(factory cmdutil.Factory) error {
	// Create the mapping from the CustomResourceDefinision GroupKind.
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return err
	}
	mapping, err := mapper.RESTMapping(crdGroupKind)
	if err != nil {
		return err
	}
	client, err := factory.UnstructuredClientForMapping(mapping)
	if err != nil {
		return err
	}
	// mapping contains the full GVK version, which is used to determine
	// the version of the ResourceGroup CRD to create. We have defined the
	// v1beta1 and v1 versions of the apiextensions group of the CRD.
	version := mapping.GroupVersionKind.Version
	klog.V(4).Infof("using apiextensions.k8s.io version: %s", version)
	rgCRDStr, ok := resourceGroupCRDs[version]
	if !ok {
		klog.V(4).Infof("ResourceGroup CRD version %s not found", version)
		return err
	}
	crd, err := stringToUnstructured(rgCRDStr)
	if err != nil {
		return err
	}
	// Set the "last-applied-annotation" so future applies work correctly.
	if err := util.CreateApplyAnnotation(crd, unstructured.UnstructuredJSONScheme); err != nil {
		return err
	}
	// Apply the CRD to the cluster and ignore already exists error.
	var clearResourceVersion = false
	var emptyNamespace = ""
	helper := resource.NewHelper(client, mapping)
	klog.V(4).Infoln("applying ResourceGroup CRD...")
	_, err = helper.Create(emptyNamespace, clearResourceVersion, crd)
	return err
}

// stringToUnstructured transforms a single resource represented by
// the passed string into a pointer to an "Unstructured" object,
// or an error if one occurred.
func stringToUnstructured(str string) (*unstructured.Unstructured, error) {
	node, err := yaml.Parse(str)
	if err != nil {
		return nil, err
	}
	s, err := node.String()
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: m}, nil
}

// resourceGroupCRDs maps the apiextensions version to the ResourceGroup
// custom resource definition string.
var resourceGroupCRDs = map[string]string{
	"v1beta1": v1beta1RGCrd,
	"v1":      v1RGCrd,
}

// ResourceGroup custom resource definition using v1beta1 version
// of the apiextensions.k8s.io API group. APIServers version 1.15
// or less will use this apiextensions group by default.
var v1beta1RGCrd = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: resourcegroups.kpt.dev
spec:
  group: kpt.dev
  names:
    kind: ResourceGroup
    listKind: ResourceGroupList
    plural: resourcegroups
    singular: resourcegroup
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: ResourceGroup is the Schema for the resourcegroups API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase.
            More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: spec defines the desired state of ResourceGroup
          properties:
            descriptor:
              description: descriptor regroups the information and metadata about
                a resource group
              properties:
                description:
                  description: description is a brief description of a group of resources
                  type: string
                links:
                  description: links are a list of descriptive URLs intended to be
                    used to surface additional information
                  items:
                    properties:
                      description:
                        description: description explains the purpose of the link
                        type: string
                      url:
                        description: url is the URL of the link
                        type: string
                    required:
                    - description
                    - url
                    type: object
                  type: array
                revision:
                  description: revision is an optional revision for a group of resources
                  type: string
                type:
                  description: type can contain prefix, such as Application/WordPress
                    or Service/Spanner
                  type: string
              type: object
            resources:
              description: resources contains a list of resources that form the resource
                group
              items:
                description: each item organizes and stores the identifying information
                  for an object. This struct (as a string) is stored in a grouping
                  object to keep track of sets of applied objects.
                properties:
                  group:
                    type: string
                  kind:
                    type: string
                  name:
                    type: string
                  namespace:
                    type: string
                required:
                - group
                - kind
                - name
                - namespace
                type: object
              type: array
            subgroups:
              description: subgroups contains a list of sub groups that the current
                group includes.
              items:
                description: each item organizes and stores the identifying information
                  for a ResourceGroup object. It includes name and namespace.
                properties:
                  name:
                    type: string
                  namespace:
                    type: string
                required:
                - name
                - namespace
                type: object
              type: array
          type: object
        status:
          description: status defines the observed state of ResourceGroup
          properties:
            conditions:
              description: conditions lists the conditions of the current status for
                the group
              items:
                properties:
                  lastTransitionTime:
                    description: last time the condition transit from one status to
                      another
                    format: date-time
                    type: string
                  message:
                    description: human-readable message indicating details about last
                      transition
                    type: string
                  reason:
                    description: one-word CamelCase reason for the condition’s last
                      transition
                    type: string
                  status:
                    description: status of the condition
                    type: string
                  type:
                    description: type of the condition
                    type: string
                required:
                - status
                - type
                type: object
              type: array
            observedGeneration:
              description: observedGeneration is the most recent generation observed.
                It corresponds to the Object's generation, which is updated on mutation
                by the API Server. Everytime the controller does a successful reconcile,
                it sets observedGeneration to match ResourceGroup.metadata.generation.
              format: int64
              type: integer
            resourceStatuses:
              description: resourceStatuses lists the status for each resource in
                the group
              items:
                description: each item contains the status of a given resource uniquely
                  identified by its group, kind, name and namespace.
                properties:
                  conditions:
                    items:
                      properties:
                        lastTransitionTime:
                          description: last time the condition transit from one status
                            to another
                          format: date-time
                          type: string
                        message:
                          description: human-readable message indicating details about
                            last transition
                          type: string
                        reason:
                          description: one-word CamelCase reason for the condition’s
                            last transition
                          type: string
                        status:
                          description: status of the condition
                          type: string
                        type:
                          description: type of the condition
                          type: string
                      required:
                      - status
                      - type
                      type: object
                    type: array
                  group:
                    type: string
                  kind:
                    type: string
                  name:
                    type: string
                  namespace:
                    type: string
                  status:
                    description: Status describes the status of a resource
                    type: string
                required:
                - group
                - kind
                - name
                - namespace
                - status
                type: object
              type: array
            subgroupStatuses:
              description: subgroupStatuses lists the status for each subgroup.
              items:
                description: each item contains the status of a given group uniquely
                  identified by its name and namespace.
                properties:
                  conditions:
                    items:
                      properties:
                        lastTransitionTime:
                          description: last time the condition transit from one status
                            to another
                          format: date-time
                          type: string
                        message:
                          description: human-readable message indicating details about
                            last transition
                          type: string
                        reason:
                          description: one-word CamelCase reason for the condition’s
                            last transition
                          type: string
                        status:
                          description: status of the condition
                          type: string
                        type:
                          description: type of the condition
                          type: string
                      required:
                      - status
                      - type
                      type: object
                    type: array
                  name:
                    type: string
                  namespace:
                    type: string
                  status:
                    description: Status describes the status of a resource
                    type: string
                required:
                - name
                - namespace
                - status
                type: object
              type: array
          required:
          - observedGeneration
          type: object
      type: object
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`

// ResourceGroup custom resource definition using v1 version
// of the apiextensions.k8s.io API group. APIServers at 1.16
// or greater will use this apiextensions group by default.
var v1RGCrd = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: resourcegroups.kpt.dev
spec:
  conversion:
    strategy: None
  group: kpt.dev
  names:
    kind: ResourceGroup
    listKind: ResourceGroupList
    plural: resourcegroups
    singular: resourcegroup
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: ResourceGroup is the Schema for the resourcegroups API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: spec defines the desired state of ResourceGroup
            properties:
              descriptor:
                description: descriptor regroups the information and metadata about
                  a resource group
                properties:
                  description:
                    description: description is a brief description of a group of resources
                    type: string
                  links:
                    description: links are a list of descriptive URLs intended to be
                      used to surface additional information
                    items:
                      properties:
                        description:
                          description: description explains the purpose of the link
                          type: string
                        url:
                          description: url is the URL of the link
                          type: string
                      required:
                      - description
                      - url
                      type: object
                    type: array
                  revision:
                    description: revision is an optional revision for a group of resources
                    type: string
                  type:
                    description: type can contain prefix, such as Application/WordPress
                      or Service/Spanner
                    type: string
                type: object
              resources:
                description: resources contains a list of resources that form the resource
                  group
                items:
                  description: each item organizes and stores the identifying information
                    for an object. This struct (as a string) is stored in a grouping
                    object to keep track of sets of applied objects.
                  properties:
                    group:
                      type: string
                    kind:
                      type: string
                    name:
                      type: string
                    namespace:
                      type: string
                  required:
                  - group
                  - kind
                  - name
                  - namespace
                  type: object
                type: array
              subgroups:
                description: subgroups contains a list of sub groups that the current
                  group includes.
                items:
                  description: Each item organizes and stores the identifying information
                    for a ResourceGroup object. It includes name and namespace.
                  properties:
                    name:
                      type: string
                    namespace:
                      type: string
                  required:
                  - name
                  - namespace
                  type: object
                type: array
            type: object
          status:
            description: status defines the observed state of ResourceGroup
            properties:
              conditions:
                description: conditions lists the conditions of the current status for
                  the group
                items:
                  properties:
                    lastTransitionTime:
                      description: last time the condition transit from one status to
                        another
                      format: date-time
                      type: string
                    message:
                      description: human-readable message indicating details about last
                        transition
                      type: string
                    reason:
                      description: one-word CamelCase reason for the condition’s last
                        transition
                      type: string
                    status:
                      description: status of the condition
                      type: string
                    type:
                      description: type of the condition
                      type: string
                  required:
                  - status
                  - type
                  type: object
                type: array
              observedGeneration:
                description: observedGeneration is the most recent generation observed.
                  It corresponds to the Object's generation, which is updated on mutation
                  by the API Server. Everytime the controller does a successful reconcile,
                  it sets observedGeneration to match ResourceGroup.metadata.generation.
                format: int64
                type: integer
              resourceStatuses:
                description: resourceStatuses lists the status for each resource in
                  the group
                items:
                  description: each item contains the status of a given resource uniquely
                    identified by its group, kind, name and namespace.
                  properties:
                    conditions:
                      items:
                        properties:
                          lastTransitionTime:
                            description: last time the condition transit from one status
                              to another
                            format: date-time
                            type: string
                          message:
                            description: human-readable message indicating details about
                              last transition
                            type: string
                          reason:
                            description: one-word CamelCase reason for the condition’s
                              last transition
                            type: string
                          status:
                            description: status of the condition
                            type: string
                          type:
                            description: type of the condition
                            type: string
                        required:
                        - status
                        - type
                        type: object
                      type: array
                    group:
                      type: string
                    kind:
                      type: string
                    name:
                      type: string
                    namespace:
                      type: string
                    status:
                      description: Status describes the status of a resource
                      type: string
                  required:
                  - group
                  - kind
                  - name
                  - namespace
                  - status
                  type: object
                type: array
              subgroupStatuses:
                description: subgroupStatuses lists the status for each subgroup.
                items:
                  description: Each item contains the status of a given group uniquely
                    identified by its name and namespace.
                  properties:
                    conditions:
                      items:
                        properties:
                          lastTransitionTime:
                            description: last time the condition transit from one status
                              to another
                            format: date-time
                            type: string
                          message:
                            description: human-readable message indicating details about
                              last transition
                            type: string
                          reason:
                            description: one-word CamelCase reason for the condition’s
                              last transition
                            type: string
                          status:
                            description: status of the condition
                            type: string
                          type:
                            description: type of the condition
                            type: string
                        required:
                        - status
                        - type
                        type: object
                      type: array
                    name:
                      type: string
                    namespace:
                      type: string
                    status:
                      description: Status describes the status of a resource
                      type: string
                  required:
                  - name
                  - namespace
                  - status
                  type: object
                type: array
            required:
            - observedGeneration
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`
