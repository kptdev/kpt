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

package fake

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
)

// Printer implements the printer.Printer interface and just ignores
// all print calls.
type Printer struct {
	outStream io.Writer
	errStream io.Writer
}

func (np *Printer) PrintPackage(*pkg.Pkg, bool) {}

func (np *Printer) OptPrintf(*printer.Options, string, ...interface{}) {}

func (np *Printer) Printf(string, ...interface{}) {}

func (np *Printer) OutStream() io.Writer { return np.outStream }

func (np *Printer) ErrStream() io.Writer { return np.errStream }

// CtxWithDefaultPrinter returns a new context with the printer which has os streams
func CtxWithDefaultPrinter() context.Context {
	return CtxWithPrinter(io.Discard, io.Discard)
}

// CtxWithPrinter returns a new context with Printer added.
func CtxWithPrinter(outStream, errStream io.Writer) context.Context {
	ctx := context.Background()
	return printer.WithContext(ctx, printer.New(outStream, errStream))
}
