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

package fake

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
)

// NilPrinter implements the printer.Printer interface and just ignores
// all print calls.
type NilPrinter struct{}

func (np *NilPrinter) PrintPackage(*pkg.Pkg, bool) {}

func (np *NilPrinter) OptPrintf(*printer.Options, string, ...interface{}) {}

func (np *NilPrinter) Printf(string, ...interface{}) {}

// CtxWithNilPrinter returns a new context with the NilPrinter added.
func CtxWithNilPrinter() context.Context {
	ctx := context.Background()
	return printer.WithContext(ctx, &NilPrinter{})
}
