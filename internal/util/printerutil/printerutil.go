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

package printerutil

import (
	"context"

	"github.com/kptdev/kpt/pkg/printer"
)

// PrintFnResultInfo displays information about the function results file.
func PrintFnResultInfo(ctx context.Context, resultsFile string, withNewLine bool) {
	pr := printer.FromContextOrDie(ctx)
	if resultsFile != "" {
		if withNewLine {
			pr.Printf("\n")
		}
		pr.Printf("For complete results, see %s\n", resultsFile)
	}
}
