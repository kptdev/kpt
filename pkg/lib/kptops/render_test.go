// Copyright 2022, 2025-2026 The kpt Authors
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

package kptops

import (
	"bytes"
	"flag"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/pkg/printer"
	"k8s.io/klog/v2"
)

func TestPackagePrinter(t *testing.T) {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "false")

	var buf bytes.Buffer
	klog.SetOutput(&buf)

	p := &packagePrinter{}

	testPkg := &pkg.Pkg{DisplayPath: "test/path"}
	p.PrintPackage(testPkg, false)

	opt := &printer.Options{
		PkgDisplayPath: "display/path",
	}
	p.OptPrintf(opt, ": Hello %s", "World")

	klog.Flush()

	got := buf.String()

	if !strings.Contains(got, `Package: "test/path"`) {
		t.Errorf("PrintPackage output missing:\n%s", got)
	}
	if !strings.Contains(got, `Package: "display/path": Hello World`) {
		t.Errorf("OptPrintf output missing:\n%s", got)
	}
}

func TestPrinterLoggingDepth(t *testing.T) {
	_ = flag.Set("logtostderr", "false")

	var buf bytes.Buffer
	klog.SetOutput(&buf)

	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		t.Errorf("call to goruntime.Caller failed")
	}
	expectedFile := filepath.Base(filename)

	p := &packagePrinter{}

	tests := []struct {
		name string
		fn   func()
	}{
		{"Printf", func() { p.Printf("Printf test: %d", 42) }},
		{"OptPrintf", func() { p.OptPrintf(&printer.Options{}, "OptPrintf test: %d", 42) }},
		{"PrintPackage", func() { p.PrintPackage(&pkg.Pkg{}, false) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()
			test.fn()
			klog.Flush()
			got := buf.String()
			if !strings.Contains(got, expectedFile) {
				t.Errorf("%s depth incorrect, expected %s in: %s", test.name, expectedFile, got)
			}
		})
	}
}
