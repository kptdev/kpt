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

	return rr.Execute(printer.WithContext(ctx, print{}))
}

type print struct{}

var _ printer.Printer = &print{}

func (p print) PrintPackage(pkg *pkg.Pkg, leadingNewline bool) {

}

func (p print) Printf(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

func (p print) OptPrintf(opt *printer.Options, format string, args ...interface{}) {

}

func (p print) OutStream() io.Writer {
	return os.Stdout
}

func (p print) ErrStream() io.Writer {
	return os.Stderr
}
