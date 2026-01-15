// Copyright 2025 The kpt and Nephio Authors
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
	"bytes"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/pkg"
)

func TestOptPrintf_WithDisplayPath(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	opt := NewOpt().PkgDisplay("my/display/path")
	pr.OptPrintf(opt, ": operation completed\n")

	expected := "Package: \"my/display/path\": operation completed\n"

	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestOptPrintf_WithUniquePath(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	opt := NewOpt().Pkg("my/unique/path")
	pr.OptPrintf(opt, ": sync successful\n")

	// RelativePath may fail, so fallback to absolute path
	expected := "Package: \"my/unique/path\": sync successful\n"

	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestOptPrintf_NilOptions(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	pr.OptPrintf(nil, "General message\n")

	expected := "General message\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestPrintPackage_WithLeadingNewline(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	p := &pkg.Pkg{DisplayPath: "my/package/path"}
	pr.PrintPackage(p, true)

	expected := "\nPackage: \"my/package/path\"\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestPrintPackage_WithoutLeadingNewline(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	p := &pkg.Pkg{DisplayPath: "another/package"}
	pr.PrintPackage(p, false)

	expected := "Package: \"another/package\"\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}
