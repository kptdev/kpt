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

package printer

import (
	"context"
	"fmt"
	"maps"
	"strings"
)

//region Builder

type Builder struct {
	Logger MapPrinter
	Args   map[string]any
}

func NewBuilder() *Builder {
	b := &Builder{}
	b.EnsureDefaults()
	return b
}

func (p *Builder) EnsureDefaults() {
	if p.Logger == nil {
		p.Logger = &StdLogger{}
	}

	if p.Args == nil {
		p.Args = map[string]any{}
	}
}

func (p *Builder) WithArg(name string, value any) *Builder {
	p.Args[name] = value
	return p
}

func (p *Builder) WithArgMap(args map[string]any) *Builder {
	maps.Copy(p.Args, args)
	return p
}

func (p *Builder) WithLogger(logger MapPrinter) *Builder {
	p.Logger = logger
	return p
}

func (p *Builder) Build() *Printer2 {
	p.EnsureDefaults()

	return &Printer2{
		l:    p.Logger,
		args: p.Args,
	}
}

//endregion

//region Printer

type Printer2 struct {
	l MapPrinter

	args map[string]any
}

func (p *Printer2) Clone() *Builder {
	return &Builder{
		Logger: p.l,
		Args:   maps.Clone(p.args),
	}
}

//region Basic Print Methods

func (p *Printer2) Print(args ...any) {
	p.l.PrintM(strings.Join(stringifyAnySlice(args), " "), p.args)
}

func (p *Printer2) Printf(format string, args ...any) {
	p.l.PrintM(fmt.Sprintf(format, args...), p.args)
}

//endregion

//region Runner-specific Methods

func (p *Printer2) Running() {
	p.Print("[RUNNING]")
}

func (p *Printer2) Fail() {
	p.Print("[FAIL]")
}

func (p *Printer2) Pass() {
	p.Print("[PASS]")
}

func (p *Printer2) Skipped() {
	p.Print("[SKIPPED]")
}

func (p *Printer2) Skippedf(format string, args ...any) {
	p.Printf("[SKIPPED] "+format, args...)
}

// TODO: decide what to do with printFnResult, printFnExecErr and printFnStderr
//       as they are multiline outputs and appending the structured args does not look correct

//region Context

const printer2Key contextKey = 1

func WithContext2(ctx context.Context, pr *Printer2) context.Context {
	return context.WithValue(ctx, printer2Key, pr)
}

func FromContext2(ctx context.Context) (*Printer2, bool) {
	pr, ok := ctx.Value(printer2Key).(*Printer2)
	return pr, ok
}

func FromContext2OrDie(ctx context.Context) *Printer2 {
	pr, ok := FromContext2(ctx)
	if !ok {
		panic("printer missing from context")
	}
	return pr
}

//endregion

//endregion

func stringifyAnySlice(a []any) []string {
	output := make([]string, 0, len(a))
	for _, v := range a {
		output = append(output, fmt.Sprintf("%s", v))
	}
	return output
}

//endregion
