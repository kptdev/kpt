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

// Package update returns a kpt updating function to a user
package update

import (
	internalupdate "github.com/kptdev/kpt/internal/util/update"
	updatetypes "github.com/kptdev/kpt/pkg/lib/update/updatetypes"
)

func GetUpdater(strategy string) updatetypes.Updater {
	return internalupdate.GetUpdater(strategy)
}
