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

package cmdutil

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"k8s.io/kubectl/pkg/cmd/util"
)

// TODO(mortent): Combine this with the internal/util/cmdutil. Also consider
// moving commands into a cmd package.

// InstallResourceGroupCRD will install the ResourceGroup CRD into the cluster.
// The function will block until the CRD is either installed and established, or
// an error was encountered.
// If the CRD could not be installed, an error of the type
// ResourceGroupCRDInstallError will be returned.
func InstallResourceGroupCRD(ctx context.Context, f util.Factory) error {
	pr := printer.FromContextOrDie(ctx)
	pr.Printf("installing inventory ResourceGroup CRD.\n")
	err := (&live.ResourceGroupInstaller{
		Factory: f,
	}).InstallRG(ctx)
	if err != nil {
		return &ResourceGroupCRDInstallError{
			Err: err,
		}
	}
	return nil
}

// ResourceGroupCRDInstallError is an error that will be returned if the
// ResourceGroup CRD can't be applied to the cluster.
type ResourceGroupCRDInstallError struct {
	Err error
}

func (*ResourceGroupCRDInstallError) Error() string {
	return "error installing ResourceGroup crd"
}

func (i *ResourceGroupCRDInstallError) Unwrap() error {
	return i.Err
}

// VerifyResourceGroupCRD verifies that the ResourceGroupCRD exists in
// the cluster. If it doesn't an error of type NoResourceGroupCRDError
// was returned.
func VerifyResourceGroupCRD(f util.Factory) error {
	if !live.ResourceGroupCRDApplied(f) {
		return &NoResourceGroupCRDError{}
	}
	return nil
}

// NoResourceGroupCRDError is an error type that will be used when a
// cluster doesn't have the ResourceGroup CRD installed.
type NoResourceGroupCRDError struct{}

func (*NoResourceGroupCRDError) Error() string {
	return "type ResourceGroup not found"
}

// ResourceGroupCRDNotLatestError is an error type that will be used when a
// cluster has a ResourceGroup CRD that doesn't match the
// latest declaration.
type ResourceGroupCRDNotLatestError struct {
	Err error
}

func (e *ResourceGroupCRDNotLatestError) Error() string {
	return fmt.Sprintf(
		"Type ResourceGroup CRD needs update. Please make sure you have the permission "+
			"to update CRD then run `kpt live install-resource-group`.\n %v", e.Err)
}
