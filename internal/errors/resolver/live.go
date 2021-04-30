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
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&liveErrorResolver{})
}

const (
	noInventoryObjError = `
Error: Package uninitialized. Please run "kpt live init" command.

The package needs to be initialized to generate the template
which will store state for resource sets. This state is
necessary to perform functionality such as deleting an entire
package or automatically deleting omitted resources (pruning).
`
	multipleInventoryObjError = `
Error: Package has multiple inventory object templates.

The package should have one and only one inventory object template.
`
	//nolint:lll
	timeoutError = `
Error: Timeout after {{printf "%.0f" .err.Timeout.Seconds}} seconds waiting for {{printf "%d" (len .err.TimedOutResources)}} out of {{printf "%d" (len .err.Identifiers)}} resources to reach condition {{ .err.Condition}}:{{ printf "\n" }}

{{- range .err.TimedOutResources}}
{{printf "%s/%s %s %s" .Identifier.GroupKind.Kind .Identifier.Name .Status .Message }}
{{- end}}
`

	TimeoutErrorExitCode = 3
)

// liveErrorResolver is an implementation of the ErrorResolver interface
// that can resolve error types used in the live functionality.
type liveErrorResolver struct{}

func (*liveErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	tmplArgs := map[string]interface{}{
		"err": err,
	}
	switch err.(type) {
	case *inventory.NoInventoryObjError:
		return ResolvedResult{
			Message: ExecuteTemplate(noInventoryObjError, tmplArgs),
		}, true
	case *inventory.MultipleInventoryObjError:
		return ResolvedResult{
			Message: ExecuteTemplate(multipleInventoryObjError, tmplArgs),
		}, true
	case *taskrunner.TimeoutError:
		return ResolvedResult{
			Message:  ExecuteTemplate(timeoutError, tmplArgs),
			ExitCode: TimeoutErrorExitCode,
		}, true
	default:
		return ResolvedResult{}, false
	}
}
