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

package resolver

import (
	"github.com/GoogleContainerTools/kpt/internal/cmdliveinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/errors"
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

	resourceGroupCRDInstallErrorMsg = `
Error: Unable to install the ResourceGroup CRD.

{{- if gt (len .cause) 0 }}
{{ printf "\nDetails:" }}
{{ printf "%s" .cause }}
{{- end }}
`
	//nolint:lll
	noResourceGroupCRDMsg = `
Error: The ResourceGroup CRD was not found in the cluster. Please install it either by using the '--install-resource-group' flag or the 'kpt live install-resource-group' command.
`

	//nolint:lll
	invInfoAlreadyExistsMsg = `
Error: Inventory information has already been added to the package Kptfile. Changing it after a package has been applied to the cluster can lead to undesired results. Use the --force flag to suppress this error.
`

	//nolint:lll
	invInfoInRGAlreadyExistsMsg = `
Error: Inventory information has already been added to the package ResourceGroup object. Changing it after a package has been applied to the cluster can lead to undesired results. Use the --force flag to suppress this error.
`

	//nolint:lll
	invInfoInKfAlreadyExistsMsg = `
Error: Inventory information has already been added to the package Kptfile object. Please consider migrating to a standalone resourcegroup object using the 'kpt live migrate' command.
`

	multipleInvInfoMsg = `
Error: Multiple Kptfile resources with inventory information found. Please make sure at most one Kptfile resource contains inventory information.
`

	//nolint:lll
	inventoryInfoValidationMsg = `
Error: The inventory information is not valid. Please update the information in the Kptfile or provide information with the command line flags. To generate the inventory information the first time, use the 'kpt live init' command.

Details:
{{- range .err.Violations}}
{{printf "%s" .Reason }}
{{- end}}
`

	unknownTypesMsg = `
Error: {{ printf "%d" (len .err.GroupVersionKinds) }} resource types could not be found in the cluster or as CRDs among the applied resources.

Resource types:
{{- range .err.GroupVersionKinds}}
{{ printf "%s" .String }}
{{- end}}
`
)

// liveErrorResolver is an implementation of the ErrorResolver interface
// that can resolve error types used in the live functionality.
type liveErrorResolver struct{}

func (*liveErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var noInventoryObjError *inventory.NoInventoryObjError
	if errors.As(err, &noInventoryObjError) {
		return ResolvedResult{
			Message: ExecuteTemplate(noInventoryObjErrorMsg, map[string]interface{}{
				"err": *noInventoryObjError,
			}),
		}, true
	}

	var multipleInventoryObjError *inventory.MultipleInventoryObjError
	if errors.As(err, &multipleInventoryObjError) {
		return ResolvedResult{
			Message: ExecuteTemplate(multipleInventoryObjErrorMsg, map[string]interface{}{
				"err": *multipleInventoryObjError,
			}),
		}, true
	}

	var resourceGroupCRDInstallError *cmdutil.ResourceGroupCRDInstallError
	if errors.As(err, &resourceGroupCRDInstallError) {
		return ResolvedResult{
			Message: ExecuteTemplate(resourceGroupCRDInstallErrorMsg, map[string]interface{}{
				"cause": resourceGroupCRDInstallError.Err.Error(),
			}),
		}, true
	}

	var noResourceGroupCRDError *cmdutil.NoResourceGroupCRDError
	if errors.As(err, &noResourceGroupCRDError) {
		return ResolvedResult{
			Message: ExecuteTemplate(noResourceGroupCRDMsg, map[string]interface{}{
				"err": *noResourceGroupCRDError,
			}),
		}, true
	}

	var invExistsError *cmdliveinit.InvExistsError
	if errors.As(err, &invExistsError) {
		return ResolvedResult{
			Message: ExecuteTemplate(invInfoAlreadyExistsMsg, map[string]interface{}{
				"err": *invExistsError,
			}),
		}, true
	}

	var invInfoInRGAlreadyExistsError *cmdliveinit.InvInRGExistsError
	if errors.As(err, &invInfoInRGAlreadyExistsError) {
		return ResolvedResult{
			Message: ExecuteTemplate(invInfoInRGAlreadyExistsMsg, map[string]interface{}{
				"err": *invInfoInRGAlreadyExistsError,
			}),
		}, true
	}

	var invInKfExistsError *cmdliveinit.InvInKfExistsError
	if errors.As(err, &invInKfExistsError) {
		return ResolvedResult{
			Message: ExecuteTemplate(invInfoInKfAlreadyExistsMsg, map[string]interface{}{
				"err": *invInKfExistsError,
			}),
		}, true
	}

	var multipleInvInfoError *live.MultipleInventoryInfoError
	if errors.As(err, &multipleInvInfoError) {
		return ResolvedResult{
			Message: ExecuteTemplate(multipleInvInfoMsg, map[string]interface{}{
				"err": *multipleInvInfoError,
			}),
		}, true
	}

	var inventoryInfoValidationError *live.InventoryInfoValidationError
	if errors.As(err, &inventoryInfoValidationError) {
		return ResolvedResult{
			Message: ExecuteTemplate(inventoryInfoValidationMsg, map[string]interface{}{
				"err": *inventoryInfoValidationError,
			}),
		}, true
	}

	var unknownTypesError *manifestreader.UnknownTypesError
	if errors.As(err, &unknownTypesError) {
		return ResolvedResult{
			Message: ExecuteTemplate(unknownTypesMsg, map[string]interface{}{
				"err": *unknownTypesError,
			}),
		}, true
	}

	var resultError *common.ResultError
	if errors.As(err, &resultError) {
		return ResolvedResult{
			Message:  resultError.Error(),
			ExitCode: 3,
		}, true
	}

	return ResolvedResult{}, false
}
