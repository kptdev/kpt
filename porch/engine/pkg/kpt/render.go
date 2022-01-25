// Copyright 2022 Google LLC
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

package kpt

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/render"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func NewRenderer() fn.Renderer {
	return &renderer{}
}

type renderer struct {
}

var _ fn.Renderer = &renderer{}

func (r *renderer) Render(ctx context.Context, pkg filesys.FileSystem, opts fn.RenderOptions) error {
	rr := render.Renderer{
		PkgPath:    opts.PkgPath,
		Runtime:    opts.Runtime,
		FileSystem: pkg,
	}

	return rr.Execute(printer.WithContext(ctx, &packagePrinter{}))
}

type packagePrinter struct{}

var _ printer.Printer = &packagePrinter{}

func (p *packagePrinter) PrintPackage(pkg *pkg.Pkg, leadingNewline bool) {
	p.Printf("Package %q: ", pkg.DisplayPath)
}

func (p *packagePrinter) Printf(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

func (p *packagePrinter) OptPrintf(opt *printer.Options, format string, args ...interface{}) {
	if opt == nil {
		p.Printf(format, args...)
		return
	}
	var prefix string
	if !opt.PkgDisplayPath.Empty() {
		prefix = fmt.Sprintf("Package %q: ", string(opt.PkgDisplayPath))
	} else if !opt.PkgPath.Empty() {
		prefix = fmt.Sprintf("Package %q: ", string(opt.PkgPath))
	}
	p.Printf(prefix+format, args...)
}

func (p *packagePrinter) OutStream() io.Writer {
	return os.Stdout
}

func (p *packagePrinter) ErrStream() io.Writer {
	return os.Stderr
}
