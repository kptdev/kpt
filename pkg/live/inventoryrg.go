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
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/cli-utils/pkg/apis/actuation"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	applyRGTimeout      = 10 * time.Second
	applyRGPollInterval = 2 * time.Second
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
	inv           *unstructured.Unstructured
	objMetas      []object.ObjMetadata
	objStatus     map[object.ObjMetadata]actuation.ObjectStatus
	strategy      inventory.Strategy
	resourceCount int
}

func (irg *InventoryResourceGroup) Strategy() inventory.Strategy {
	return irg.strategy
}

var _ inventory.Storage = &InventoryResourceGroup{}
var _ inventory.Info = &InventoryResourceGroup{}

// WrapInventoryObj returns a closure that takes a passed ResourceGroup (as a resource.Info),
// wraps it with the InventoryResourceGroup and upcasts the wrapper as
// a Storage interface. The number of managed resources per ResourceGroup is specified
// by the resourceCount argument.
// Strategy is hardcoded to be of NameStrategy as logic for sharded ResourceGroups
// needs to be handled within kpt, and using label strategy would cause errors in
// cli-utils.
func WrapInventoryObj(resourceCount int) func(*unstructured.Unstructured) inventory.Storage {
	return func(obj *unstructured.Unstructured) inventory.Storage {
		if obj != nil {
			klog.V(4).Infof("wrapping Inventory obj: %s/%s\n", obj.GetNamespace(), obj.GetName())
		}
		return &InventoryResourceGroup{inv: obj, strategy: inventory.NameStrategy, resourceCount: resourceCount}
	}
}

// WrapInventoryInfoObj takes a passed ResourceGroup (as a resource.Info),
// wraps it with the InventoryResourceGroup and upcasts the wrapper as
// an Inventory interface.
// Strategy is passed as a function argument as there are use cases where using the label strategy
// is useful, eg. mass deletion of all sharded ResourceGroups. In all other cases, the NameStrategy should
// be used instead.
func WrapInventoryInfoObj(obj *unstructured.Unstructured, strategy inventory.Strategy) inventory.Info {
	if obj != nil {
		klog.V(4).Infof("wrapping InventoryInfo obj: %s/%s\n", obj.GetNamespace(), obj.GetName())
	}
	return &InventoryResourceGroup{inv: obj, strategy: strategy}
}

func InvToUnstructuredFunc(inv inventory.Info) *unstructured.Unstructured {
	switch invInfo := inv.(type) {
	case *InventoryResourceGroup:
		return invInfo.inv
	default:
		return nil
	}
}

// Name(), Namespace(), and ID() are InventoryResourceGroup functions to
// implement the InventoryInfo interface.
func (irg *InventoryResourceGroup) Name() string {
	return irg.inv.GetName()
}

func (irg *InventoryResourceGroup) Namespace() string {
	return irg.inv.GetNamespace()
}

func (irg *InventoryResourceGroup) ID() string {
	labels := irg.inv.GetLabels()
	if val, found := labels[common.InventoryLabel]; found {
		return val
	}
	return ""
}

// Load is an Inventory interface function returning the set of
// object metadata from the wrapped ResourceGroup, or an error.
func (irg *InventoryResourceGroup) Load() (object.ObjMetadataSet, error) {
	objs := object.ObjMetadataSet{}
	if irg.inv == nil {
		return objs, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("loading inventory...")
	items, exists, err := unstructured.NestedSlice(irg.inv.Object, "spec", "resources")
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
		objMeta := object.ObjMetadata{
			GroupKind: groupKind,
			Name:      name,
			Namespace: namespace,
		}
		objs = append(objs, objMeta)
	}
	return objs, nil
}

// ReadResourceGroupObjects creates a list of managed GKNN ObjectMetadata as an ObjectMetadataSet from an unstructured ResourceGroup.
func (irg *InventoryResourceGroup) ReadResourceGroupObjects(rgObj *unstructured.Unstructured) (object.ObjMetadataSet, error) {
	var objs object.ObjMetadataSet
	klog.V(4).Infof("loading inventory...")

	items, exists, err := unstructured.NestedSlice(rgObj.Object, "spec", "resources")
	if err != nil {
		err := fmt.Errorf("error retrieving object metadata from inventory object")
		return objs, err
	}
	if !exists {
		klog.V(4).Infof("Inventory (spec.resources) does not exist")
		return objs, nil
	}

	klog.V(4).Infof("loading %d inventory items", len(items))
	for _, item := range items {
		objMeta, err := idFromUnstructuredField(item)
		if err != nil {
			return nil, err
		}

		objs = append(objs, objMeta)
	}

	return objs, nil
}

// Store is an Inventory interface function implemented to store
// the object metadata in the wrapped ResourceGroup. Actual storing
// happens in "GetObject".
func (irg *InventoryResourceGroup) Store(objMetas object.ObjMetadataSet, status []actuation.ObjectStatus) error {
	irg.objMetas = objMetas
	irg.objStatus = make(map[object.ObjMetadata]actuation.ObjectStatus, len(status))
	for _, s := range status {
		irg.objStatus[inventory.ObjMetadataFromObjectReference(s.ObjectReference)] = s
	}

	return nil
}

// GetObject returns the wrapped object (ResourceGroup) as a resource.Info
// or an error if one occurs.
func (irg *InventoryResourceGroup) GetObject() (*unstructured.Unstructured, error) {
	if irg.inv == nil {
		return nil, fmt.Errorf("inventory info is nil")
	}
	objStatusMap := map[object.ObjMetadata]actuation.ObjectStatus{}
	for _, s := range irg.objStatus {
		objStatusMap[inventory.ObjMetadataFromObjectReference(s.ObjectReference)] = s
	}
	klog.V(4).Infof("getting inventory resource group")
	// Create a slice of Resources as empty Interface
	klog.V(4).Infof("Creating list of %d resources", len(irg.objMetas))
	var objs []interface{}
	for _, objMeta := range irg.objMetas {
		klog.V(4).Infof("storing inventory obj refercence: %s/%s", objMeta.Namespace, objMeta.Name)
		objs = append(objs, idToUnstructuredMap(objMeta))
	}
	klog.V(4).Infof("Creating list of %d resources status", len(irg.objMetas))
	var objStatus []interface{}
	for _, objMeta := range irg.objMetas {
		status, found := objStatusMap[objMeta]
		if found {
			klog.V(4).Infof("storing inventory obj refercence and its status: %s/%s", objMeta.Namespace, objMeta.Name)
			objStatus = append(objStatus, map[string]interface{}{
				"group":     objMeta.GroupKind.Group,
				"kind":      objMeta.GroupKind.Kind,
				"namespace": objMeta.Namespace,
				"name":      objMeta.Name,
				"status":    "Unknown",
				"strategy":  status.Strategy.String(),
				"actuation": status.Actuation.String(),
				"reconcile": status.Reconcile.String(),
			})
		}
	}

	// Create the inventory object by copying the template.
	invCopy := irg.inv.DeepCopy()
	// Adds or clears the inventory ObjMetadata to the ResourceGroup "spec.resources" section
	if len(objs) == 0 {
		klog.V(4).Infoln("clearing inventory resources")
		unstructured.RemoveNestedField(invCopy.UnstructuredContent(),
			"spec", "resources")
		unstructured.RemoveNestedField(invCopy.UnstructuredContent(),
			"status", "resourceStatuses")
	} else {
		klog.V(4).Infof("storing inventory (%d) resources", len(objs))
		err := unstructured.SetNestedSlice(invCopy.UnstructuredContent(),
			objs, "spec", "resources")
		if err != nil {
			return nil, err
		}
		err = unstructured.SetNestedSlice(invCopy.UnstructuredContent(),
			objStatus, "status", "resourceStatuses")
		if err != nil {
			return nil, err
		}
		generation := invCopy.GetGeneration()
		err = unstructured.SetNestedField(invCopy.UnstructuredContent(),
			generation, "status", "observedGeneration")
		if err != nil {
			return nil, err
		}
	}
	return invCopy, nil
}

// sortResourceGroups is a utility function for sorting a list of sharded ResourceGroup objects by the
// ResourceGroup object name.
func sortResourceGroups(resourcegroups []unstructured.Unstructured) {
	sort.Slice(resourcegroups, func(i, j int) bool {
		return resourcegroups[i].GetName() < resourcegroups[j].GetName()
	})
}

// Apply is a Storage interface function implemented to apply the inventory
// object.
func (irg *InventoryResourceGroup) Apply(dc dynamic.Interface, mapper meta.RESTMapper, statusPolicy inventory.StatusPolicy) error {
	namespacedClient, err := irg.getNamespacedClient(dc, mapper)
	if err != nil {
		return err
	}

	// Get all cluster ResourceGroup objects using label selector.
	resourceList, err := namespacedClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: irg.shardedLabel(),
	})
	if err != nil {
		return err
	}
	resourcegroups := resourceList.Items

	// Internal flag to indicate whether current ResourceGroup should be updated to include new labels.
	var shouldUpdate bool

	// Fallback to using name lookup for backwards compatibility where a ResourceGroup already exists on cluster without
	// the label selector. This occurs when the on cluster ResourceGroup was created with an older version of kpt.
	if len(resourcegroups) == 0 {
		rg, err := namespacedClient.Get(context.TODO(), irg.Name(), metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if rg != nil {
			resourcegroups = []unstructured.Unstructured{*rg}
			shouldUpdate = true
		}
	}

	if len(resourcegroups) == 0 {
		// Create cluster ResourceGroup object(s) if it does not exist on cluster.
		return irg.createResourceGroups(dc, mapper, statusPolicy)
	}

	sortResourceGroups(resourcegroups)

	// Create a map that stores all managaged objects to determine if we have newer objects to be added
	// to the ResourceGroup. We also store the index position to be able to store statuses as well by referencing
	// the status slice index.
	managedResources := make(map[object.ObjMetadata]struct{}, len(irg.objMetas))
	for _, objMeta := range irg.objMetas {
		managedResources[objMeta] = struct{}{}
	}

	var lastRG *unstructured.Unstructured
	for _, rg := range resourcegroups {
		rg := rg
		lastRG = &rg
		objs, err := irg.ReadResourceGroupObjects(&rg)
		if err != nil {
			return err
		}

		for _, obj := range objs {
			delete(managedResources, obj)
		}
	}

	// All resources in local ResourceGroup is already being tracked on cluster.
	if len(managedResources) == 0 {
		if !shouldUpdate {
			return nil
		}

		// Need to update existing ResourceGroup on-cluster to add label.
		irg.addShardedLabel(lastRG)

		_, err = namespacedClient.Update(context.TODO(), lastRG, metav1.UpdateOptions{})
		return err
	}

	// Simply append managed resources to the cluster's last ResourceGroup object in the
	// sharded series as it should be able to store the additional resources.
	if irg.shardingDisabled() || getObjCount(lastRG) <= irg.resourceCount {
		items, _, _ := unstructured.NestedSlice(lastRG.Object, "spec", "resources")
		statuses, _, _ := unstructured.NestedSlice(lastRG.Object, "status", "resourceStatuses")
		for obj := range managedResources {
			items = append(items, idToUnstructuredMap(obj))
			statuses = append(statuses, statusToUnstructuredMap(irg.objStatus[obj]))
			delete(managedResources, obj)

			// Stop appending to item list if we need to shard to a new ResourceGroup.
			if len(items) == irg.resourceCount && !irg.shardingDisabled() {
				break
			}
		}

		err = unstructured.SetNestedSlice(lastRG.Object, items, "spec", "resources")
		if err != nil {
			return err
		}

		// Add sharded label in case we need to update on-cluster ResourceGroup.
		irg.addShardedLabel(lastRG)
		appliedObj, err := namespacedClient.Update(context.TODO(), lastRG, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			if err := updateStatus(namespacedClient, lastRG, statuses, appliedObj); err != nil {
				return err
			}
		}
	}

	// Create newer ResourceGroups to keep storing more resources when sharding is enabled, and
	// previous ResourceGroup is full.
	for len(managedResources) > 0 {
		newRG := irg.nextOf(lastRG)
		items := make([]interface{}, 0, min(irg.resourceCount, len(managedResources)))
		statuses := make([]interface{}, 0, min(irg.resourceCount, len(managedResources)))
		for obj := range managedResources {
			items = append(items, idToUnstructuredMap(obj))
			statuses = append(statuses, statusToUnstructuredMap(irg.objStatus[obj]))
			delete(managedResources, obj)

			if len(items) == irg.resourceCount {
				break
			}
		}
		err = unstructured.SetNestedSlice(newRG.Object, items, "spec", "resources")
		if err != nil {
			return err
		}

		appliedObj, err := namespacedClient.Create(context.TODO(), newRG, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			if err := updateStatus(namespacedClient, newRG, statuses, appliedObj); err != nil {
				return err
			}
		}

		lastRG = newRG
	}

	return nil
}

// shardedName generates the formatted name in a series of sharded ResourceGroups.
// The name should be formatted as `<resourcegroup_name>-<name-length-hash>-<index>`.
// An example sharded name is: "resourcegroup-13-1"
func (irg *InventoryResourceGroup) shardedName(idx int) string {
	return fmt.Sprintf("%s-%d-%d", irg.Name(), len(irg.Name()), idx)
}

// shardedLabel genereates the formatted label name required for sharded ResourceGroups.
// The label value should be formatted as `<resourcegroup_name>/id`.
func (irg *InventoryResourceGroup) shardedLabel() string {
	return fmt.Sprintf("%s/id", irg.Name())
}

func (irg *InventoryResourceGroup) shardingDisabled() bool {
	return irg.resourceCount <= 0
}

// nextOf returns a new unstructured object with metadata that attributes it to the
// next sharded ResourceGroup object.
func (irg *InventoryResourceGroup) nextOf(curr *unstructured.Unstructured) *unstructured.Unstructured {
	newRG := &unstructured.Unstructured{}
	newRG.SetGroupVersionKind(ResourceGroupGVK)
	newRG.SetNamespace(curr.GetNamespace())

	// Add the required labels.
	newRG.SetLabels(curr.GetLabels())

	// Determine name of new ResourceGroup.
	name := curr.GetName()
	// Append the `<name_length_hash>-<index>` suffix to generate the next name in the series.
	if name == irg.Name() {
		newRG.SetName(irg.shardedName(1))
		return newRG
	}

	// Extract the index of the current sharded ResourceGroup to be incremented on.
	str := strings.TrimPrefix(name, fmt.Sprintf("%s-%d-", irg.Name(), len(irg.Name())))
	currID, _ := strconv.Atoi(str)
	newRG.SetName(irg.shardedName(currID + 1))
	return newRG
}

// getObjCount returns the number of resources stored within the given ResourceGroup object.
func getObjCount(rg *unstructured.Unstructured) int {
	items, exists, err := unstructured.NestedSlice(rg.Object, "spec", "resources")
	if err != nil || !exists {
		return 0
	}

	return len(items)
}

// ApplyWithPrune applies the inventory objects with pruning logic.
func (irg *InventoryResourceGroup) ApplyWithPrune(dc dynamic.Interface, mapper meta.RESTMapper,
	statusPolicy inventory.StatusPolicy, allObjs object.ObjMetadataSet) error {
	namespacedClient, err := irg.getNamespacedClient(dc, mapper)
	if err != nil {
		return err
	}

	// Get all cluster ResourceGroup objects using label selector.
	resourceList, err := namespacedClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: irg.shardedLabel(),
	})
	if err != nil {
		return err
	}
	resourcegroups := resourceList.Items

	// Fallback to using name lookup for backwards compatibility where a ResourceGroup already exists on cluster without
	// the label selector. This occurs when the on cluster ResourceGroup was created with an older version of kpt.
	if len(resourcegroups) == 0 {
		rg, err := namespacedClient.Get(context.TODO(), irg.Name(), metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if rg != nil {
			resourcegroups = []unstructured.Unstructured{*rg}
		}
	}

	managedResources := make(map[object.ObjMetadata]struct{}, len(allObjs))
	for _, id := range allObjs {
		managedResources[id] = struct{}{}
	}

	needCollapse := false
	var duplicateToBalance []interface{}
	for _, rg := range resourcegroups {
		items, exists, err := unstructured.NestedSlice(rg.Object, "spec", "resources")
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		var newItems []interface{}
		var newStatuses []interface{}
		for _, item := range items {
			id, err := idFromUnstructuredField(item)
			if err != nil {
				return err
			}

			if _, exists := managedResources[id]; exists {
				newItems = append(newItems, item)
				newStatuses = append(newStatuses, statusToUnstructuredMap(irg.objStatus[id]))
			} else {
				needCollapse = true
			}
		}

		err = unstructured.SetNestedSlice(rg.Object, newItems, "spec", "resources")
		if err != nil {
			return err
		}

		// Add sharded label in the event pruning occurs on an older ResourceGroup without the label.
		irg.addShardedLabel(&rg)

		// Flag when we pruned, but the ResourceGroup sharding limit has actually decreased,
		// causing this ResourceGroup to be over the max allowed sharding limit.
		if len(newItems) > irg.resourceCount && !irg.shardingDisabled() {
			duplicateToBalance = append(duplicateToBalance, newItems...)
			needCollapse = true
		}

		appliedObj, err := namespacedClient.Update(context.TODO(), &rg, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			err := updateStatus(namespacedClient, &rg, newStatuses, appliedObj)
			if err != nil {
				return err
			}
		}
	}

	// Create newer ResourceGroups if sharding limit has decreased.
	if len(duplicateToBalance) > 0 {
		// Ensure all on-cluster ResourceGroups do not have more resources than allowed as determined by resourceCount.
		err := irg.enforceShardingLimit(allObjs, duplicateToBalance, resourcegroups, namespacedClient, statusPolicy)
		if err != nil {
			return err
		}
	}

	if needCollapse {
		// Reshuffle GKNNs to the start of ResourceGroup object series.
		return irg.collapse(namespacedClient, statusPolicy)
	}

	return nil
}

// enforceShardingLimit runs goes through each ResourceGroup and ensures that all ResourceGroups do not contain more managed GKNNs
// than the specified sharding limit count. This is done by going through a list of ObjectMetadataSet to determine which GKNNs need to be
// removed from ResourceGroups that have too many tracked resources and needs to be split off into another ResourceGroup.
func (irg *InventoryResourceGroup) enforceShardingLimit(allObjs object.ObjMetadataSet,
	duplicateToBalance []interface{}, resourcegroups []unstructured.Unstructured,
	namespacedClient dynamic.ResourceInterface, statusPolicy inventory.StatusPolicy) error {
	managedResources := make(map[object.ObjMetadata]struct{}, len(allObjs))
	for _, item := range duplicateToBalance {
		id, err := idFromUnstructuredField(item)
		if err != nil {
			return err
		}
		managedResources[id] = struct{}{}
	}
	lastRG := &resourcegroups[len(resourcegroups)-1]

	// Create new ReasourceGroups to house duplicate managed GKNNs so we can safely
	// delete them in earlier ResourceGroups that do not obey the maximum sharding limit.
	for len(managedResources) > 0 {
		newRG := irg.nextOf(lastRG)
		items := make([]interface{}, 0, min(irg.resourceCount, len(managedResources)))
		statuses := make([]interface{}, 0, min(irg.resourceCount, len(managedResources)))
		for obj := range managedResources {
			items = append(items, idToUnstructuredMap(obj))
			statuses = append(statuses, statusToUnstructuredMap(irg.objStatus[obj]))
			delete(managedResources, obj)

			if len(items) == irg.resourceCount {
				break
			}
		}
		err := unstructured.SetNestedSlice(newRG.Object, items, "spec", "resources")
		if err != nil {
			return err
		}

		appliedObj, err := namespacedClient.Create(context.TODO(), newRG, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			if err := updateStatus(namespacedClient, newRG, statuses, appliedObj); err != nil {
				return err
			}
		}

		lastRG = newRG
	}

	// Loop through all created ResourceGroups and delete the excess managed GKNNs.
	resourceList, err := namespacedClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: irg.shardedLabel(),
	})
	if err != nil {
		return err
	}

	resourcegroups = resourceList.Items
	for _, rg := range resourcegroups {
		items, exists, err := unstructured.NestedSlice(rg.Object, "spec", "resources")
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		if len(items) <= irg.resourceCount {
			continue
		}

		items = items[:irg.resourceCount]
		err = unstructured.SetNestedSlice(rg.Object, items, "spec", "resources")
		if err != nil {
			return err
		}
		appliedObj, err := namespacedClient.Update(context.TODO(), &rg, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			currStatuses, err := irg.getStatuses(items)
			if err != nil {
				return err
			}
			err = updateStatus(namespacedClient, &rg, currStatuses, appliedObj)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// collapse the set of ResourceGroups into as few ResourceGroups as possible, given the
// MaxResourceCount setting.
func (irg *InventoryResourceGroup) collapse(namespacedClient dynamic.ResourceInterface, statusPolicy inventory.StatusPolicy) error {
	if irg.shardingDisabled() {
		klog.V(4).Info("skipping re-balancing of ResourceGroups as sharding is disabled")
		return nil
	}

	// Get all cluster ResourceGroup objects using label selector.
	resourceList, err := namespacedClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: irg.shardedLabel(),
	})
	if err != nil {
		return err
	}

	resourcegroups := resourceList.Items
	sortResourceGroups(resourcegroups)

	// Get all GKNNs across all sharded ResourceGroups.
	var items []interface{}
	var seen object.ObjMetadataSet
	for _, rg := range resourcegroups {
		resources, exists, err := unstructured.NestedSlice(rg.Object, "spec", "resources")
		if err != nil || !exists {
			continue
		}

		for _, obj := range resources {
			id, err := idFromUnstructuredField(obj)
			if err != nil {
				return err
			}

			if seen.Contains(id) {
				continue
			}

			items = append(items, obj)
			seen = append(seen, id)
		}
	}

	count := len(items)

	// Re-balance GKNN objects in sharded ResourceGroups.
	objIdx := 0
	for objIdx < count {
		var currItems []interface{}

		limitIdx := objIdx + irg.resourceCount
		rgIdx := objIdx / irg.resourceCount
		currRG := resourcegroups[rgIdx]

		if limitIdx < len(items) {
			currItems = items[objIdx:limitIdx]
		} else {
			currItems = items[objIdx:]
		}

		// Get the required statuses.
		currStatuses, err := irg.getStatuses(currItems)
		if err != nil {
			return err
		}

		err = unstructured.SetNestedSlice(currRG.Object, currItems, "spec", "resources")
		if err != nil {
			return err
		}
		appliedObj, err := namespacedClient.Update(context.TODO(), &currRG, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			err := updateStatus(namespacedClient, &currRG, currStatuses, appliedObj)
			if err != nil {
				return err
			}
		}

		objIdx += irg.resourceCount
	}

	// Delete ResourceGroups that contain duplicated GKNNs.
	for rgIdx := len(resourcegroups) - 1; rgIdx >= objIdx/irg.resourceCount; rgIdx-- {
		// Do not delete the first ResourceGroup in series. If we do reach rgIdx == 0, it indicates that
		// kpt is not managing any GKNNs in its inventory.
		if rgIdx == 0 {
			break
		}

		err = namespacedClient.Delete(context.TODO(), resourcegroups[rgIdx].GetName(), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// getStatuses iterates through a list of unstructured GKNNs and returns a slice of statuses for the found objects.
func (irg *InventoryResourceGroup) getStatuses(objMetas []interface{}) ([]interface{}, error) {
	statuses := make([]interface{}, 0, len(objMetas))

	for _, objMeta := range objMetas {
		item, err := idFromUnstructuredField(objMeta)
		if err != nil {
			return nil, err
		}

		s, exists := irg.objStatus[item]
		if !exists {
			continue
		}

		statuses = append(statuses, statusToUnstructuredMap(s))
	}

	return statuses, nil
}

// createResourceGroups creates new ResourceGroup object(s) on cluster, and handles the sharding of managed
// resources into multiple ResourceGroup objects.
func (irg *InventoryResourceGroup) createResourceGroups(dc dynamic.Interface, mapper meta.RESTMapper, statusPolicy inventory.StatusPolicy) error {
	namespacedClient, err := irg.getNamespacedClient(dc, mapper)
	if err != nil {
		return err
	}

	// totalStored tracks the total number of resources stored within all sharded ResourceGroups.
	totalStored := 0

	// Create ResourceGroups by partitioning the GKNNs to be stored, or create an empty
	// ResourceGroup if we no GKNNs are being tracked.
	for totalStored < len(irg.objMetas) || totalStored == 0 {
		var items []interface{}
		var statuses []interface{}

		switch irg.shardingDisabled() {
		case true:
			for _, objMeta := range irg.objMetas {
				items = append(items, idToUnstructuredMap(objMeta))
			}

			if statusPolicy == inventory.StatusPolicyAll {
				for _, objStatus := range irg.objStatus {
					statuses = append(statuses, statusToUnstructuredMap(objStatus))
				}
			}
		default: // Split managed objects into current ResourceGroup.
			for partitionCount := 0; partitionCount < irg.resourceCount && totalStored+partitionCount < len(irg.objMetas); partitionCount++ {
				objMeta := irg.objMetas[totalStored+partitionCount]
				items = append(items, idToUnstructuredMap(objMeta))

				if statusPolicy == inventory.StatusPolicyAll {
					objStatus := statusToUnstructuredMap(irg.objStatus[objMeta])
					statuses = append(statuses, objStatus)
				}
			}
		}

		var rg *unstructured.Unstructured
		switch totalStored {
		case 0: // Initial RG in series.
			rg = irg.inv
			rg.SetResourceVersion("")
		default:
			rg = irg.inv.DeepCopy()
			rg.SetResourceVersion("")
			rg.SetName(irg.shardedName(totalStored / irg.resourceCount))
		}

		irg.addShardedLabel(rg)

		err := unstructured.SetNestedSlice(rg.Object, items, "spec", "resources")
		if err != nil {
			return err
		}

		appliedObj, err := namespacedClient.Create(context.TODO(), rg, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		if statusPolicy == inventory.StatusPolicyAll {
			if err := updateStatus(namespacedClient, rg, statuses, appliedObj); err != nil {
				return err
			}
		}

		totalStored += len(items)

		// Exit loop if the ResourceGroup is empty.
		if len(items) == 0 {
			break
		}
	}

	return nil
}

// updateStatus will update resourceStatuses on a live ResourceGroup on cluster by setting the required fields in the unstructured object.
func updateStatus(namespacedClient dynamic.ResourceInterface, rg *unstructured.Unstructured, statuses []interface{}, appliedObj *unstructured.Unstructured) error {
	switch len(statuses) {
	case 0:
		unstructured.RemoveNestedField(rg.Object, "status", "resourceStatuses")
	default:
		err := unstructured.SetNestedSlice(rg.Object, statuses, "status", "resourceStatuses")
		if err != nil {
			return fmt.Errorf("unable to set nested field for resourceStatuses: %w", err)
		}
	}

	// Update ResourceGroup to have the latest ResourceVersion and Generation.
	rg.SetResourceVersion(appliedObj.GetResourceVersion())
	generation := appliedObj.GetGeneration()
	err := unstructured.SetNestedField(rg.Object,
		generation, "status", "observedGeneration")
	if err != nil {
		return fmt.Errorf("unable to set nested field for .status.observedGeneration: %w", err)
	}

	_, err = namespacedClient.UpdateStatus(context.TODO(), rg, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update status: %w", err)
	}

	return nil
}

// getNamespacedClient returns a namespaced client for interacting with ResourceGroups on cluster.
func (irg *InventoryResourceGroup) getNamespacedClient(dc dynamic.Interface, mapper meta.RESTMapper) (dynamic.ResourceInterface, error) {
	invInfo, err := irg.GetObject()
	if err != nil {
		return nil, err
	}
	if invInfo == nil {
		return nil, errors.New("attempting to interact with a nil inventory object")
	}

	mapping, err := mapper.RESTMapping(invInfo.GroupVersionKind().GroupKind(), invInfo.GroupVersionKind().Version)
	if err != nil {
		return nil, fmt.Errorf("unable to create rest mapping for ResourceGroup: %w", err)
	}

	// Create client to interact with cluster.
	namespacedClient := dc.Resource(mapping.Resource).Namespace(invInfo.GetNamespace())

	return namespacedClient, nil
}

// idFromUnstructuredField serializes a nested field of stored GKNN resources within an unstructured ResourceGroup object,
// to an ObjectMetadata.
func idFromUnstructuredField(fields interface{}) (object.ObjMetadata, error) {
	item, ok := fields.(map[string]interface{})
	if !ok {
		return object.ObjMetadata{}, fmt.Errorf("unable to cast field to the required map type for: %v", fields)
	}

	namespace, _, err := unstructured.NestedString(item, "namespace")
	if err != nil {
		return object.ObjMetadata{}, err
	}
	name, _, err := unstructured.NestedString(item, "name")
	if err != nil {
		return object.ObjMetadata{}, err
	}
	group, _, err := unstructured.NestedString(item, "group")
	if err != nil {
		return object.ObjMetadata{}, err
	}
	kind, _, err := unstructured.NestedString(item, "kind")
	if err != nil {
		return object.ObjMetadata{}, err
	}
	groupKind := schema.GroupKind{
		Group: strings.TrimSpace(group),
		Kind:  strings.TrimSpace(kind),
	}

	id := object.ObjMetadata{
		GroupKind: groupKind,
		Name:      name,
		Namespace: namespace,
	}

	return id, nil
}

// idToUnstructuredMap serializes a given ObjMetadata into an unstructured interface of
// GKNN to be tracked in a ResourceGroup.
func idToUnstructuredMap(obj object.ObjMetadata) map[string]interface{} {
	return map[string]interface{}{
		"group":     obj.GroupKind.Group,
		"kind":      obj.GroupKind.Kind,
		"namespace": obj.Namespace,
		"name":      obj.Name,
	}
}

// statusToUnstructuredMap serializes a given ObjMetadata into an unstructured interface of
// GKNN to be tracked in a ResourceGroup.
func statusToUnstructuredMap(status actuation.ObjectStatus) map[string]interface{} {
	return map[string]interface{}{
		"group":     status.Group,
		"kind":      status.Kind,
		"namespace": status.Namespace,
		"name":      status.Name,
		"status":    "Unknown",
		"actuation": status.Actuation.String(),
		"reconcile": status.Reconcile.String(),
		"strategy":  status.Strategy.String(),
	}
}

// addShardedLabel adds a new label to the ResourceGroup object.
func (irg *InventoryResourceGroup) addShardedLabel(resourcegroup *unstructured.Unstructured) {
	labels := resourcegroup.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[irg.shardedLabel()] = ""
	resourcegroup.SetLabels(labels)
}

// min returns the minimum of 2 integers.
func min(i, j int) int {
	if i < j {
		return i
	}

	return j
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

// ResourceGroupCRDApplied returns true if the inventory ResourceGroup
// CRD is available from the current RESTMapper, or false otherwise.
func ResourceGroupCRDApplied(factory cmdutil.Factory) bool {
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		klog.V(4).Infof("error retrieving RESTMapper when checking ResourceGroup CRD: %s\n", err)
		return false
	}
	_, err = mapper.RESTMapping(ResourceGroupGVK.GroupKind())
	if err != nil {
		klog.V(7).Infof("error retrieving ResourceGroup RESTMapping: %s\n", err)
		return false
	}
	return true
}

// ResourceGroupCRDMatched checks if the ResourceGroup CRD
// in the cluster matches the CRD in the kpt binary.
func ResourceGroupCRDMatched(factory cmdutil.Factory) bool {
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		klog.V(4).Infof("error retrieving RESTMapper when checking ResourceGroup CRD: %s\n", err)
		return false
	}
	crd, err := rgCRD(mapper)
	if err != nil {
		klog.V(7).Infof("failed to get ResourceGroup CRD from string: %s", err)
		return false
	}

	dc, err := factory.DynamicClient()
	if err != nil {
		klog.V(7).Infof("error getting the dynamic client: %s\n", err)
		return false
	}

	mapping, err := mapper.RESTMapping(crdGroupKind)
	if err != nil {
		klog.V(7).Infof("Failed to get mapping of CRD type: %s", err)
		return false
	}

	liveCRD, err := dc.Resource(mapping.Resource).Get(context.TODO(), "resourcegroups.kpt.dev", metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			APIVersion: crd.GetAPIVersion(),
			Kind:       "CustomResourceDefinition",
		},
	})
	if err != nil {
		klog.V(7).Infof("error getting the ResourceGroup CRD from cluster: %s\n", err)
		return false
	}

	liveSpec, _, err := unstructured.NestedMap(liveCRD.Object, "spec")
	if err != nil {
		klog.V(7).Infof("error getting the ResourceGroup CRD spec from cluster: %s\n", err)
		return false
	}
	latestspec, _, err := unstructured.NestedMap(crd.Object, "spec")
	if err != nil {
		klog.V(7).Infof("error getting the ResourceGroup CRD spec from string: %s\n", err)
		return false
	}
	return reflect.DeepEqual(liveSpec, latestspec)
}

// ResourceGroupInstaller can install the ResourceGroup CRD into a cluster.
type ResourceGroupInstaller struct {
	Factory cmdutil.Factory
}

func (rgi *ResourceGroupInstaller) InstallRG(ctx context.Context) error {
	poller, err := status.NewStatusPoller(rgi.Factory)
	if err != nil {
		return err
	}

	mapper, err := rgi.Factory.ToRESTMapper()
	if err != nil {
		return err
	}

	crd, err := rgCRD(mapper)
	if err != nil {
		return err
	}

	if err := rgi.applyRG(crd); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	objs := object.UnstructuredSetToObjMetadataSet([]*unstructured.Unstructured{crd})
	ctx, cancel := context.WithTimeout(ctx, applyRGTimeout)
	return func() error {
		defer cancel()
		for e := range poller.Poll(ctx, objs, polling.PollOptions{PollInterval: applyRGPollInterval}) {
			switch e.Type {
			case pollevent.ErrorEvent:
				return e.Error
			case pollevent.ResourceUpdateEvent:
				if e.Resource.Status == kstatus.CurrentStatus {
					meta.MaybeResetRESTMapper(mapper)
				}
			}
		}
		return nil
	}()
}

func (rgi *ResourceGroupInstaller) applyRG(crd runtime.Object) error {
	mapper, err := rgi.Factory.ToRESTMapper()
	if err != nil {
		return err
	}
	mapping, err := mapper.RESTMapping(crdGroupKind)
	if err != nil {
		return err
	}
	client, err := rgi.Factory.UnstructuredClientForMapping(mapping)
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
	_, err = helper.Create(emptyNamespace, clearResourceVersion, crd)
	return err
}

// rgCRD returns the ResourceGroup CRD in Unstructured format or an error.
func rgCRD(mapper meta.RESTMapper) (*unstructured.Unstructured, error) {
	mapping, err := mapper.RESTMapping(crdGroupKind)
	if err != nil {
		return nil, err
	}
	// mapping contains the full GVK version, which is used to determine
	// the version of the ResourceGroup CRD to create. We have defined the
	// v1beta1 and v1 versions of the apiextensions group of the CRD.
	version := mapping.GroupVersionKind.Version
	klog.V(4).Infof("using apiextensions.k8s.io version: %s", version)
	rgCRDStr, ok := resourceGroupCRDs[version]
	if !ok {
		klog.V(4).Infof("ResourceGroup CRD version %s not found", version)
		return nil, err
	}
	crd, err := stringToUnstructured(rgCRDStr)
	if err != nil {
		return nil, err
	}
	return crd, nil
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
          description: ResourceGroupSpec defines the desired state of ResourceGroup
          properties:
            descriptor:
              description: Descriptor regroups the information and metadata about
                a resource group
              properties:
                description:
                  description: Description is a brief description of a group of resources
                  type: string
                links:
                  description: Links are a list of descriptive URLs intended to be
                    used to surface additional information
                  items:
                    properties:
                      description:
                        description: Description explains the purpose of the link
                        type: string
                      url:
                        description: Url is the URL of the link
                        type: string
                    required:
                    - description
                    - url
                    type: object
                  type: array
                revision:
                  description: Revision is an optional revision for a group of resources
                  type: string
                type:
                  description: Type can contain prefix, such as Application/WordPress
                    or Service/Spanner
                  type: string
              type: object
            resources:
              description: Resources contains a list of resources that form the resource group
              items:
                description: ObjMetadata organizes and stores the identifying information
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
          type: object
        status:
          description: ResourceGroupStatus defines the observed state of ResourceGroup
          properties:
            conditions:
              description: Conditions lists the conditions of the current status for
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
                    description: one-word CamelCase reason for the condition's last
                      transition
                    type: string
                  status:
                    description: Status of the condition
                    type: string
                  type:
                    description: Type of the condition
                    type: string
                required:
                - status
                - type
                type: object
              type: array
            observedGeneration:
              description: ObservedGeneration is the most recent generation observed.
                It corresponds to the Object's generation, which is updated on mutation
                by the API Server. Everytime the controller does a successful reconcile,
                it sets ObservedGeneration to match ResourceGroup.metadata.generation.
              format: int64
              type: integer
            resourceStatuses:
              description: ResourceStatuses lists the status for each resource in
                the group
              items:
                description: ResourceStatus contains the status of a given resource
                  uniquely identified by its group, kind, name and namespace.
                properties:
                  actuation:
                    description: actuation indicates whether actuation has been
                      performed yet and how it went.
                    type: string
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
                          description: Status of the condition
                          type: string
                        type:
                          description: Type of the condition
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
                  reconcile:
                    description: reconcile indicates whether reconciliation has
                      been performed yet and how it went.
                    type: string
                  sourceHash:
                    type: string
                  status:
                    description: Status describes the status of a resource
                    type: string
                  strategy:
                    description: strategy indicates the method of actuation (apply
                      or delete) used or planned to be used.
                    type: string
                required:
                - group
                - kind
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
            description: ResourceGroupSpec defines the desired state of ResourceGroup
            properties:
              descriptor:
                description: Descriptor regroups the information and metadata about
                  a resource group
                properties:
                  description:
                    description: Description is a brief description of a group of
                      resources
                    type: string
                  links:
                    description: Links are a list of descriptive URLs intended to
                      be used to surface additional information
                    items:
                      properties:
                        description:
                          description: Description explains the purpose of the link
                          type: string
                        url:
                          description: Url is the URL of the link
                          type: string
                      required:
                      - description
                      - url
                      type: object
                    type: array
                  revision:
                    description: Revision is an optional revision for a group of resources
                    type: string
                  type:
                    description: Type can contain prefix, such as Application/WordPress
                      or Service/Spanner
                    type: string
                type: object
              resources:
                description: Resources contains a list of resources that form the
                  resource group
                items:
                  description: ObjMetadata organizes and stores the identifying information
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
            type: object
          status:
            description: ResourceGroupStatus defines the observed state of ResourceGroup
            properties:
              conditions:
                description: Conditions lists the conditions of the current status
                  for the group
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
                      description: one-word CamelCase reason for the condition’s last
                        transition
                      type: string
                    status:
                      description: Status of the condition
                      type: string
                    type:
                      description: Type of the condition
                      type: string
                  required:
                  - status
                  - type
                  type: object
                type: array
              observedGeneration:
                description: ObservedGeneration is the most recent generation observed.
                  It corresponds to the Object's generation, which is updated on mutation
                  by the API Server. Everytime the controller does a successful reconcile,
                  it sets ObservedGeneration to match ResourceGroup.metadata.generation.
                format: int64
                type: integer
              resourceStatuses:
                description: ResourceStatuses lists the status for each resource in
                  the group
                items:
                  description: ResourceStatus contains the status of a given resource
                    uniquely identified by its group, kind, name and namespace.
                  properties:
                    actuation:
                      description: actuation indicates whether actuation has been
                        performed yet and how it went.
                      type: string
                    conditions:
                      items:
                        properties:
                          lastTransitionTime:
                            description: last time the condition transit from one
                              status to another
                            format: date-time
                            type: string
                          message:
                            description: human-readable message indicating details
                              about last transition
                            type: string
                          reason:
                            description: one-word CamelCase reason for the condition’s
                              last transition
                            type: string
                          status:
                            description: Status of the condition
                            type: string
                          type:
                            description: Type of the condition
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
                    reconcile:
                      description: reconcile indicates whether reconciliation has
                        been performed yet and how it went.
                      type: string
                    sourceHash:
                      type: string
                    status:
                      description: Status describes the status of a resource
                      type: string
                    strategy:
                      description: strategy indicates the method of actuation (apply
                        or delete) used or planned to be used.
                      type: string
                  required:
                  - group
                  - kind
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
