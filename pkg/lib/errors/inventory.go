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

package errors

// InvExistsError defines new error when the inventory
// values have already been set on the Kptfile.
type InvExistsError struct{}

func (i *InvExistsError) Error() string {
	return "inventory information already set for package"
}

// InvInRGExistsError defines new error when the inventory
// values have already been set on the ResourceGroup file and we will warn
// the user to migrate rather than init. This is part of kpt live STDIN work.
type InvInRGExistsError struct{}

func (i *InvInRGExistsError) Error() string {
	return "inventory information already set for package"
}

// InvInKfExistsError defines new error when the inventory
// values have already been set on the Kptfile and we will warn
// the user to migrate rather than init. This is part of kpt live STDIN work.
type InvInKfExistsError struct{}

func (i *InvInKfExistsError) Error() string {
	return "inventory information already set within Kptfile for package"
}
