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
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
)

func TestPackagePrinter(t *testing.T) {
	t.Run("PrintPackage without leading newline", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		testPkg := &pkg.Pkg{
			DisplayPath: "test/path",
		}

		p.PrintPackage(testPkg, false)

		output := errBuf.String()
		assert.Contains(t, output, "test/path")
		assert.NotContains(t, output, "\n\nPackage")
	})

	t.Run("PrintPackage with leading newline", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		testPkg := &pkg.Pkg{
			DisplayPath: "test/path",
		}

		p.PrintPackage(testPkg, true)

		output := errBuf.String()
		assert.Contains(t, output, "test/path")
		assert.Contains(t, output, "\nPackage")
	})

	t.Run("Printf", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		p.Printf("test message")
		assert.Contains(t, errBuf.String(), "test message")

		errBuf.Reset()
		p.Printf("test message with args: %s %d", "hello", 42)
		assert.Contains(t, errBuf.String(), "test message with args: hello 42")
	})

	t.Run("OptPrintf with nil options", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		p.OptPrintf(nil, "test message")
		assert.Contains(t, errBuf.String(), "test message")

		errBuf.Reset()
		p.OptPrintf(nil, "test with args: %s", "value")
		assert.Contains(t, errBuf.String(), "test with args: value")
	})

	t.Run("OptPrintf with PkgDisplayName", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		opt := printer.NewOpt().PkgName("my-package")

		p.OptPrintf(opt, "test message")
		output := errBuf.String()
		assert.Contains(t, output, "my-package")
		assert.Contains(t, output, "test message")
	})

	t.Run("OptPrintf with PkgDisplayPath", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		opt := printer.NewOpt().PkgDisplay("display/path")

		p.OptPrintf(opt, "test message")
		output := errBuf.String()
		assert.Contains(t, output, "display/path")
		assert.Contains(t, output, "test message")
	})

	t.Run("OptPrintf with PkgPath", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		opt := printer.NewOpt().Pkg("unique/path")

		p.OptPrintf(opt, "test message")
		output := errBuf.String()
		assert.Contains(t, output, "unique/path")
		assert.Contains(t, output, "test message")
	})

	t.Run("OptPrintf with multiple options set", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		opt := printer.NewOpt().
			PkgName("display-name").
			PkgDisplay("display/path").
			Pkg("unique/path")

		p.OptPrintf(opt, "test message")
		output := errBuf.String()
		assert.Contains(t, output, "display-name")
		assert.Contains(t, output, "test message")
	})

	t.Run("OutStream", func(t *testing.T) {
		var outBuf bytes.Buffer
		p := printer.New(&outBuf, io.Discard)

		outStream := p.OutStream()
		require.NotNil(t, outStream)

		_, err := io.WriteString(outStream, "test output")
		require.NoError(t, err)
		assert.Equal(t, "test output", outBuf.String())
	})

	t.Run("ErrStream", func(t *testing.T) {
		var errBuf bytes.Buffer
		p := printer.New(io.Discard, &errBuf)

		errStream := p.ErrStream()
		require.NotNil(t, errStream)

		_, err := io.WriteString(errStream, "test error")
		require.NoError(t, err)
		assert.Equal(t, "test error", errBuf.String())
	})
}

func TestPackagePrinterStub(t *testing.T) {
	t.Run("PrintPackage stub", func(t *testing.T) {
		p := &packagePrinter{}
		testPkg := &pkg.Pkg{
			DisplayPath: "test/path",
		}

		assert.NotPanics(t, func() {
			p.PrintPackage(testPkg, false)
		})

		assert.NotPanics(t, func() {
			p.PrintPackage(testPkg, true)
		})
	})

	t.Run("Printf stub", func(t *testing.T) {
		p := &packagePrinter{}

		assert.NotPanics(t, func() {
			p.Printf("test message")
		})

		assert.NotPanics(t, func() {
			p.Printf("test message with args: %s %d", "hello", 42)
		})
	})

	t.Run("OptPrintf stub with nil options", func(t *testing.T) {
		p := &packagePrinter{}

		assert.NotPanics(t, func() {
			p.OptPrintf(nil, "test message")
		})
	})

	t.Run("OptPrintf stub with options", func(t *testing.T) {
		p := &packagePrinter{}
		opt := printer.NewOpt().PkgName("my-package")

		assert.NotPanics(t, func() {
			p.OptPrintf(opt, "test message")
		})
	})

	t.Run("OutStream stub", func(t *testing.T) {
		p := &packagePrinter{}

		stream := p.OutStream()
		assert.NotNil(t, stream)
		assert.Equal(t, os.Stdout, stream)
	})

	t.Run("ErrStream stub", func(t *testing.T) {
		p := &packagePrinter{}

		stream := p.ErrStream()
		assert.NotNil(t, stream)
		assert.Equal(t, os.Stderr, stream)
	})
}

func TestPrinterLoggingDepth(t *testing.T) {
	klog.LogToStderr(false)
	defer klog.LogToStderr(true)

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
