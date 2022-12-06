package testhelpers

/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type Harness struct {
	*testing.T

	Ctx context.Context
}

func NewHarness(t *testing.T) *Harness {
	h := &Harness{
		T:   t,
		Ctx: context.Background(),
	}

	return h
}

func (h *Harness) AssertMatchesFile(p string, got string) {
	if os.Getenv("WRITE_GOLDEN_OUTPUT") != "" {
		// Short-circuit when the output is correct
		b, err := os.ReadFile(p)
		if err == nil && bytes.Equal(b, []byte(got)) {
			return
		}

		if err := os.WriteFile(p, []byte(got), 0644); err != nil {
			h.Fatalf("failed to write golden output %s: %v", p, err)
		}
		h.Errorf("wrote output to %s", p)
	} else {
		want := string(h.MustReadFile(p))
		if diff := cmp.Diff(want, got); diff != "" {
			h.Errorf("unexpected diff in %s: %s", p, diff)
		}
	}
}

func (h *Harness) MustReadFile(p string) []byte {
	b, err := os.ReadFile(p)
	if err != nil {
		h.Fatalf("error from ReadFile(%q): %v", p, err)
	}
	return b
}
