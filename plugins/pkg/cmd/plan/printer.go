// Copyright 2023 The kpt Authors
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

package plan

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
)

// printPlan writes a plan to out for human consumption (usually will be stdout)
func printPlan(ctx context.Context, plan *Plan, out io.Writer) error {
	minwidth := 0
	tabwidth := 4
	padding := 1
	padchar := byte(' ')
	flags := uint(0) //tabwriter.AlignRight | tabwriter.Debug
	w := tabwriter.NewWriter(out, minwidth, tabwidth, padding, padchar, flags)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", "ACTION", "KIND", "NAMESPACE", "NAME")
	for _, action := range plan.Spec.Actions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", action.Type, action.Kind, action.Namespace, action.Name)
	}
	w.Flush()

	return nil
}
