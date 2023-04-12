// Copyright 2021 The kpt Authors
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

package resolver

import (
	"fmt"

	initialization "github.com/GoogleContainerTools/kpt/commands/live/init"
	"github.com/GoogleContainerTools/kpt/internal/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/print/common"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&liveErrorResolver{})
}

const (
	noInventoryObjErrorMsg = `
Error: Package uninitialized. Please run "kpt live init" command.

The package needs to be initialized to generate the template
which will store state for resource sets. This state is
necessary to perform functionality such as deleting an entire
package or automatically deleting omitted resources (pruning).
`
	multipleInventoryObjErrorMsg = `
Error: Package has multiple inventory object templates.

The package should have one and only one inventory object template.
`

	//nolint:lll
	noResourceGroupCRDMsg = `
Error: The ResourceGroup CRD was not found in the cluster. Please install it either by using the '--install-resource-group' flag or the 'kpt live install-resource-group' command.
`

	//nolint:lll
	invInfoAlreadyExistsMsg = `
Error: Inventory information has already been added to the package. Changing it after a package has been applied to the cluster can lead to undesired results. Use the --force flag to suppress this error.
`

	//nolint:lll
	invInfoInRGAlreadyExistsMsg = `
Error: Inventory information has already been added to the package ResourceGroup object. Changing it after a package has been applied to the cluster can lead to undesired results. Use the --force flag to suppress this error.
`

	//nolint:lll
	invInfoInKfAlreadyExistsMsg = `
Error: Inventory information has already been added to the package Kptfile object. Please consider migrating to a standalone resourcegroup object using the 'kpt live migrate' command.
`

	multipleResourceGroupsMsg = `
Error: Multiple ResourceGroup objects found. Please make sure at most one ResourceGroup object exists within the package.
`
)

// liveErrorResolver is an implementation of the ErrorResolver interface
// that can resolve error types used in the live functionality.
type liveErrorResolver struct{}

func (*liveErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var noInventoryObjError *inventory.NoInventoryObjError
	if errors.As(err, &noInventoryObjError) {
		msg := noInventoryObjErrorMsg
		return ResolvedResult{Message: msg}, true
	}

	var multipleInventoryObjError *inventory.MultipleInventoryObjError
	if errors.As(err, &multipleInventoryObjError) {
		msg := multipleInventoryObjErrorMsg
		return ResolvedResult{Message: msg}, true
	}

	var resourceGroupCRDInstallError *cmdutil.ResourceGroupCRDInstallError
	if errors.As(err, &resourceGroupCRDInstallError) {
		msg := "Error: Unable to install the ResourceGroup CRD."

		cause := resourceGroupCRDInstallError.Err
		msg += fmt.Sprintf("\nDetails: %v", cause)

		return ResolvedResult{Message: msg}, true
	}

	var noResourceGroupCRDError *cmdutil.NoResourceGroupCRDError
	if errors.As(err, &noResourceGroupCRDError) {
		msg := noResourceGroupCRDMsg
		return ResolvedResult{Message: msg}, true
	}

	var invExistsError *initialization.InvExistsError
	if errors.As(err, &invExistsError) {
		msg := invInfoAlreadyExistsMsg
		return ResolvedResult{Message: msg}, true
	}

	var invInfoInRGAlreadyExistsError *initialization.InvInRGExistsError
	if errors.As(err, &invInfoInRGAlreadyExistsError) {
		msg := invInfoInRGAlreadyExistsMsg
		return ResolvedResult{Message: msg}, true
	}

	var invInKfExistsError *initialization.InvInKfExistsError
	if errors.As(err, &invInKfExistsError) {
		msg := invInfoInKfAlreadyExistsMsg
		return ResolvedResult{Message: msg}, true
	}

	var multipleResourceGroupsError *pkg.MultipleResourceGroupsError
	if errors.As(err, &multipleResourceGroupsError) {
		msg := multipleResourceGroupsMsg
		return ResolvedResult{Message: msg}, true
	}

	var inventoryInfoValidationError *live.InventoryInfoValidationError
	if errors.As(err, &inventoryInfoValidationError) {
		msg := "Error: The inventory information is not valid."
		msg += " Please update the information in the ResourceGroup file or provide information with the command line flags."
		msg += " To generate the inventory information the first time, use the 'kpt live init' command."

		msg += "\nDetails:\n"
		for _, v := range inventoryInfoValidationError.Violations {
			msg += fmt.Sprintf("%s\n", v.Reason)
		}

		return ResolvedResult{Message: msg}, true
	}

	var unknownTypesError *manifestreader.UnknownTypesError
	if errors.As(err, &unknownTypesError) {
		msg := fmt.Sprintf("Error: %d resource types could not be found in the cluster or as CRDs among the applied resources.", len(unknownTypesError.GroupVersionKinds))
		msg += "\n\nResource types:\n"
		for _, gvk := range unknownTypesError.GroupVersionKinds {
			msg += fmt.Sprintf("%s\n", gvk)
		}

		return ResolvedResult{Message: msg}, true
	}

	var resultError *common.ResultError
	if errors.As(err, &resultError) {
		return ResolvedResult{
			Message:  "", // Printer summary now replaces ResultError message
			ExitCode: 3,
		}, true
	}

	return ResolvedResult{}, false
}
