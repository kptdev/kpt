// Copyright 2022 The kpt Authors
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

package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormula(t *testing.T) {
	want := `# Copyright 2019 The kpt Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

class Kpt < Formula
  desc "Toolkit to manage,and apply Kubernetes Resource config data files"
  homepage "https://googlecontainertools.github.io/kpt"
  url "https://github.com/GoogleContainerTools/kpt/archive/v0.0.0-fake.tar.gz"
  sha256 "4e42c5ce1a23511405beb5f51cfe07885fa953db448265fe74ee9b81e0def277"

  depends_on "go" => :build

  def install
    ENV["GO111MODULE"] = "on"
    system "go", "build", "-ldflags", "-X github.com/GoogleContainerTools/kpt/run.version=#{version}", *std_go_args
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/kpt version")
  end
end
`

	httpClient := &http.Client{
		Transport: &fakeServer{t: t},
	}
	url := "https://github.com/GoogleContainerTools/kpt/archive/v0.0.0-fake.tar.gz"
	got, err := buildFormula(httpClient, url)
	if err != nil {
		t.Fatalf("error from buildFormula(%q): %v", url, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("buildFormula(%q) returned unexpected diff (-want +got):\n%s", url, diff)
	}
}

type fakeServer struct {
	t *testing.T
}

func (s *fakeServer) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path != "/GoogleContainerTools/kpt/archive/v0.0.0-fake.tar.gz" {
		s.t.Errorf("Expected to request '/GoogleContainerTools/kpt/archive/v0.0.0-fake.tar.gz', got: %s", req.URL.Path)
	}
	body := bytes.NewReader([]byte(`This is fake content so that our tests don't download big files`))
	return &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(body),
	}, nil
}
