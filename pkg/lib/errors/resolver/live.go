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

package resolver

import (
	"errors"
	"fmt"
	"strings"

	initialization "github.com/kptdev/kpt/commands/live/init"
	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/pkg/lib/util/cmdutil"
	"github.com/kptdev/kpt/pkg/live"
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
	if _, ok := errors.AsType[*inventory.NoInventoryObjError](err); ok {
		msg := noInventoryObjErrorMsg
		return ResolvedResult{Message: msg}, true
	}

	if _, ok := errors.AsType[*inventory.MultipleInventoryObjError](err); ok {
		msg := multipleInventoryObjErrorMsg
		return ResolvedResult{Message: msg}, true
	}

	if resourceGroupCRDInstallError, ok := errors.AsType[*cmdutil.ResourceGroupCRDInstallError](err); ok {
		msg := "Error: Unable to install the ResourceGroup CRD."

		cause := resourceGroupCRDInstallError.Err
		msg += fmt.Sprintf("\nDetails: %v", cause)

		return ResolvedResult{Message: msg}, true
	}

	if _, ok := errors.AsType[*cmdutil.NoResourceGroupCRDError](err); ok {
		msg := noResourceGroupCRDMsg
		return ResolvedResult{Message: msg}, true
	}

	if _, ok := errors.AsType[*initialization.InvExistsError](err); ok {
		msg := invInfoAlreadyExistsMsg
		return ResolvedResult{Message: msg}, true
	}

	if _, ok := errors.AsType[*initialization.InvInRGExistsError](err); ok {
		msg := invInfoInRGAlreadyExistsMsg
		return ResolvedResult{Message: msg}, true
	}

	if _, ok := errors.AsType[*initialization.InvInKfExistsError](err); ok {
		msg := invInfoInKfAlreadyExistsMsg
		return ResolvedResult{Message: msg}, true
	}

	if _, ok := errors.AsType[*pkg.MultipleResourceGroupsError](err); ok {
		msg := multipleResourceGroupsMsg
		return ResolvedResult{Message: msg}, true
	}

	if inventoryInfoValidationError, ok := errors.AsType[*live.InventoryInfoValidationError](err); ok {
		var msg strings.Builder
		msg.WriteString("Error: The inventory information is not valid.")
		msg.WriteString(" Please update the information in the ResourceGroup file or provide information with the command line flags.")
		msg.WriteString(" To generate the inventory information the first time, use the 'kpt live init' command.")

		msg.WriteString("\nDetails:\n")
		for _, v := range inventoryInfoValidationError.Violations {
			fmt.Fprintf(&msg, "%s\n", v.Reason)
		}

		return ResolvedResult{Message: msg.String()}, true
	}

	if unknownTypesError, ok := errors.AsType[*manifestreader.UnknownTypesError](err); ok {
		var msg strings.Builder
		fmt.Fprintf(&msg, "Error: %d resource types not found in the cluster or as CRDs among the applied resources.", len(unknownTypesError.GroupVersionKinds))
		msg.WriteString("\n\nResource types:\n")
		for _, gvk := range unknownTypesError.GroupVersionKinds {
			fmt.Fprintf(&msg, "%s\n", gvk)
		}

		return ResolvedResult{Message: msg.String()}, true
	}

	if _, ok := errors.AsType[*common.ResultError](err); ok {
		return ResolvedResult{
			Message:  "", // Printer summary now replaces ResultError message
			ExitCode: 3,
		}, true
	}

	return ResolvedResult{}, false
}
