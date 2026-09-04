// Copyright 2025 The kpt Authors
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

	opt := NewOpt().DisplayPath("my/display/path")
	pr.OptPrintf(opt, " operation completed\n")

	expected := "Package \"my/display/path\": operation completed\n"

	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestOptPrintf_WithUniquePath(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	opt := NewOpt().Path("my/unique/path")
	pr.OptPrintf(opt, " sync successful\n")

	// RelativePath may fail, so fallback to absolute path
	expected := "Package \"my/unique/path\": sync successful\n"

	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestOptPrintf_WithDisplayName(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	opt := NewOpt().DisplayName("my-repo.my-package.v1")
	pr.OptPrintf(opt, " sync successful\n")

	// RelativePath may fail, so fallback to absolute path
	expected := "Package \"my-repo.my-package.v1\": sync successful\n"

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

	expected := "\nPackage \"my/package/path\":\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestPrintPackage_WithoutLeadingNewline(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, &buf)

	p := &pkg.Pkg{DisplayPath: "another/package"}
	pr.PrintPackage(p, false)

	expected := "Package \"another/package\":\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestPrinter_ContextualFieldsAndEvents(t *testing.T) {
	t.Run("WithField and WithFields scoping", func(t *testing.T) {
		var buf bytes.Buffer
		pr := New(&buf, &buf)

		p1 := pr.WithField("package", "wordpress")
		p2 := p1.WithFunction("set-labels", "latest").WithFields(ContextualFields{
			"requestID": "req-123",
			"user":      "admin",
		})

		p2.PrintRunning("set-labels", 2)
		got := buf.String()
		expected := "[RUNNING] image=\"set-labels\" package=\"wordpress\" requestID=\"req-123\" resourceCount=\"2\" tag=\"latest\" user=\"admin\"\n"
		if got != expected {
			t.Errorf("Expected %q, got %q", expected, got)
		}

		// Verify original printer p1 is unmodified
		buf.Reset()
		p1.PrintRunning("set-labels", 0)
		gotP1 := buf.String()
		expectedP1 := "[RUNNING] package=\"wordpress\"\n"
		if gotP1 != expectedP1 {
			t.Errorf("Expected %q, got %q", expectedP1, gotP1)
		}
	})

	t.Run("PrintPass and PrintFail", func(t *testing.T) {
		var buf bytes.Buffer
		pr := New(&buf, &buf).WithFunction("kubeconform", "latest").WithPackage("wordpress")

		pr.PrintPass("kubeconform", 250*1000*1000) // 250ms
		gotPass := buf.String()
		expectedPass := "[PASS] image=\"kubeconform\" package=\"wordpress\" tag=\"latest\" time=\"250ms\"\n"
		if gotPass != expectedPass {
			t.Errorf("Expected %q, got %q", expectedPass, gotPass)
		}

		buf.Reset()
		pr.PrintFail("kubeconform", 100*1000*1000, nil)
		gotFail := buf.String()
		expectedFail := "[FAIL] image=\"kubeconform\" package=\"wordpress\" tag=\"latest\" time=\"100ms\"\n"
		if gotFail != expectedFail {
			t.Errorf("Expected %q, got %q", expectedFail, gotFail)
		}
	})

	t.Run("PrintSummary", func(t *testing.T) {
		var buf bytes.Buffer
		pr := New(&buf, &buf).WithField("user", "porch-controller")

		pr.PrintSummary(4, 2, 1170*1000*1000)
		gotSummary := buf.String()
		expectedSummary := "Successfully executed 4 function(s) in 2 package(s) time=\"1.17s\" user=\"porch-controller\"\n"
		if gotSummary != expectedSummary {
			t.Errorf("Expected %q, got %q", expectedSummary, gotSummary)
		}
	})
}
