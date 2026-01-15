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

package kpt

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/internal/util/render"
	"github.com/kptdev/kpt/pkg/lib/fnruntime"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func readFile(t *testing.T, path string) []byte {
	if data, err := os.ReadFile(path); err != nil {
		t.Fatalf("Cannot read file %q", err)
		return nil
	} else {
		return data
	}
}

func TestRender(t *testing.T) {
	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	if err != nil {
		t.Fatalf("Cannot compute absolute path for ./testdata: %v", err)
	}

	for _, test := range []struct {
		name string
		pkg  string
		want string
	}{
		{
			name: "render-with-function-config",
			pkg:  "simple-bucket",
			want: "expected.txt",
		},
		{
			name: "render-with-inline-config",
			pkg:  "simple-bucket",
			want: "expected.txt",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			r := render.Renderer{
				PkgPath:    filepath.Join(testdata, test.name, test.pkg),
				Runtime:    &runtime{},
				FileSystem: filesys.FileSystemOrOnDisk{},
				Output:     &output,
			}
			r.RunnerOptions.InitDefaults(fnruntime.GHCRImagePrefix)

			if _, err := r.Execute(fake.CtxWithDefaultPrinter()); err != nil {
				t.Errorf("Render failed: %v", err)
			}

			got := output.String()
			want := readFile(t, filepath.Join(testdata, test.name, test.want))

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("Unexpected result (-want, +got): %s", diff)
			}
		})
	}
}

func TestPackagePrinter(t *testing.T) {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "false")

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
	flag.Set("logtostderr", "false")

	var buf bytes.Buffer
	klog.SetOutput(&buf)

	_, filename, _, _ := goruntime.Caller(0)
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
