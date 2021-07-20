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

package live

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/strings"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// InventoryInfoValidationError is the error returned if validation of the
// inventory information fails.
type InventoryInfoValidationError struct {
	errors.ValidationError
}

func (e *InventoryInfoValidationError) Error() string {
	return fmt.Sprintf("inventory failed validation for fields: %s",
		strings.JoinStringsWithQuotes(e.Violations.Fields()))
}

// MultipleInventoryInfoError is the error returned if there are multiple
// Kptfile resources in a stream which has inventory information.
type MultipleInventoryInfoError struct{}

func (e *MultipleInventoryInfoError) Error() string {
	return "multiple Kptfiles contains inventory information"
}

// Load reads resources either from disk or from an input stream. It filters
// out resources that should be ignored and defaults the namespace for
// namespace-scoped resources that doesn't have the namespace set. It also looks
// for inventory information inside Kptfile resources.
// It returns the resources in unstructured format and the inventory information.
// If no inventory information is found, that is not considered an error here.
func Load(f util.Factory, path string, stdIn io.Reader) ([]*unstructured.Unstructured, kptfilev1.Inventory, error) {
	if path == "-" {
		return loadFromStream(f, stdIn)
	}
	return loadFromDisk(f, path)
}

// loadFromStream reads resources from the provided reader and returns the
// filtered resources and any inventory information found in Kptfile resources.
// If there is more than one Kptfile in the stream with inventory information, that
// is considered an error.
func loadFromStream(f util.Factory, r io.Reader) ([]*unstructured.Unstructured, kptfilev1.Inventory, error) {
	var stdInBuf bytes.Buffer
	tee := io.TeeReader(r, &stdInBuf)

	invInfo, err := readInvInfoFromStream(tee)
	if err != nil {
		return nil, kptfilev1.Inventory{}, err
	}

	ro, err := toReaderOptions(f)
	if err != nil {
		return nil, kptfilev1.Inventory{}, err
	}

	objs, err := (&ResourceGroupStreamManifestReader{
		ReaderName:    "stdin",
		Reader:        &stdInBuf,
		ReaderOptions: ro,
	}).Read()
	if err != nil {
		return nil, kptfilev1.Inventory{}, err
	}
	return objs, invInfo, nil
}

func readInvInfoFromStream(in io.Reader) (kptfilev1.Inventory, error) {
	invFilter := &InventoryFilter{}
	if err := (&kio.Pipeline{
		Inputs: []kio.Reader{
			&kio.ByteReader{
				Reader: in,
			},
		},
		Filters: []kio.Filter{
			kio.FilterAll(invFilter),
		},
	}).Execute(); err != nil {
		return kptfilev1.Inventory{}, err
	}

	if len(invFilter.Inventories) > 1 {
		return kptfilev1.Inventory{}, &MultipleInventoryInfoError{}
	}

	if len(invFilter.Inventories) == 1 {
		return *invFilter.Inventories[0], nil
	}
	return kptfilev1.Inventory{}, nil
}

// loadFromdisk reads resources from the provided directory and any subfolder.
// It returns the filtered resources and any inventory information found in
// Kptfile resources.
// Only the Kptfile in the root directory will be checked for inventory information.
func loadFromDisk(f util.Factory, path string) ([]*unstructured.Unstructured, kptfilev1.Inventory, error) {
	invInfo, err := readInvInfoFromDisk(path)
	if err != nil {
		return nil, kptfilev1.Inventory{}, err
	}

	ro, err := toReaderOptions(f)
	if err != nil {
		return nil, kptfilev1.Inventory{}, err
	}

	objs, err := (&ResourceGroupPathManifestReader{
		PkgPath:       path,
		ReaderOptions: ro,
	}).Read()
	if err != nil {
		return nil, kptfilev1.Inventory{}, err
	}

	return objs, invInfo, nil
}

func readInvInfoFromDisk(path string) (kptfilev1.Inventory, error) {
	p, err := pkg.New(path)
	if err != nil {
		return kptfilev1.Inventory{}, err
	}

	kf, err := p.Kptfile()
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return kptfilev1.Inventory{}, nil
	}
	if err != nil {
		return kptfilev1.Inventory{}, err
	}

	if kf.Inventory == nil {
		return kptfilev1.Inventory{}, nil
	}

	return *kf.Inventory, nil
}

// InventoryFilter is an implementation of the yaml.Filter interface
// that extracts inventory information from Kptfile resources.
type InventoryFilter struct {
	Inventories []*kptfilev1.Inventory
}

func (i *InventoryFilter) Filter(object *yaml.RNode) (*yaml.RNode, error) {
	if object.GetApiVersion() != kptfilev1.KptFileAPIVersion ||
		object.GetKind() != kptfilev1.KptFileName {
		return object, nil
	}

	s, err := object.String()
	if err != nil {
		return object, err
	}
	kf, err := pkg.DecodeKptfile(bytes.NewBufferString(s))
	if err != nil {
		return nil, err
	}
	if kf.Inventory != nil {
		i.Inventories = append(i.Inventories, kf.Inventory)
	}
	return object, nil
}

// toReaderOptions returns the readerOptions for a factory.
func toReaderOptions(f util.Factory) (manifestreader.ReaderOptions, error) {
	namespace, enforceNamespace, err := f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return manifestreader.ReaderOptions{}, err
	}
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return manifestreader.ReaderOptions{}, err
	}

	return manifestreader.ReaderOptions{
		Mapper:           mapper,
		Namespace:        namespace,
		EnforceNamespace: enforceNamespace,
	}, nil
}

// ToInventoryInfo takes the information in the provided inventory object and
// return an InventoryResourceGroup implementation of the InventoryInfo interface.
func ToInventoryInfo(inventory kptfilev1.Inventory) (inventory.InventoryInfo, error) {
	if err := validateInventory(inventory); err != nil {
		return nil, err
	}
	invObj := generateInventoryObj(inventory)
	return WrapInventoryInfoObj(invObj), nil
}

func validateInventory(inventory kptfilev1.Inventory) error {
	var violations errors.Violations
	if inventory.Name == "" {
		violations = append(violations, errors.Violation{
			Field:  "name",
			Value:  inventory.Name,
			Type:   errors.Missing,
			Reason: "\"inventory.name\" must not be empty",
		})
	}
	if inventory.Namespace == "" {
		violations = append(violations, errors.Violation{
			Field:  "namespace",
			Value:  inventory.Namespace,
			Type:   errors.Missing,
			Reason: "\"inventory.namespace\" must not be empty",
		})
	}
	if inventory.InventoryID == "" {
		violations = append(violations, errors.Violation{
			Field:  "inventoryID",
			Value:  inventory.InventoryID,
			Type:   errors.Missing,
			Reason: "\"inventory.inventoryID\" must not be empty",
		})
	}
	if len(violations) > 0 {
		return &InventoryInfoValidationError{
			ValidationError: errors.ValidationError{
				Violations: violations,
			},
		}
	}
	return nil
}

func generateInventoryObj(inv kptfilev1.Inventory) *unstructured.Unstructured {
	// Create and return ResourceGroup custom resource as inventory object.
	groupVersion := fmt.Sprintf("%s/%s", ResourceGroupGVK.Group, ResourceGroupGVK.Version)
	var inventoryObj = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": groupVersion,
			"kind":       ResourceGroupGVK.Kind,
			"metadata": map[string]interface{}{
				"name":      inv.Name,
				"namespace": inv.Namespace,
				"labels": map[string]interface{}{
					common.InventoryLabel: inv.InventoryID,
				},
			},
			"spec": map[string]interface{}{
				"resources": []interface{}{},
			},
		},
	}
	labels := inv.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[common.InventoryLabel] = inv.InventoryID
	inventoryObj.SetLabels(labels)
	inventoryObj.SetAnnotations(inv.Annotations)
	return inventoryObj
}
