// Copyright 2019 Google LLC
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
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "must specify new version\n")
		os.Exit(1)
	}
	input := Input{Version: os.Args[1]}
	var err error
	input.Sha, err = getSha(input.Version)
	if err != nil {
		os.Exit(1)
	}

	// generate the formula text
	t, err := template.New("formula").Parse(formula)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	// write the new formula
	b := &bytes.Buffer{}
	if err = t.Execute(b, input); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	err = os.WriteFile(filepath.Join("Formula", "kpt.rb"), b.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func getSha(version string) (string, error) {
	// create the dir for the data
	d, err := os.MkdirTemp("", "kpt-bin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return "", err
	}
	defer os.RemoveAll(d)

	fmt.Println(
		"fetching https://github.com/GoogleContainerTools/kpt/archive/" + version + ".tar.gz")
	// get the content
	resp, err := http.Get(
		"https://github.com/GoogleContainerTools/kpt/archive/" + version + ".tar.gz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return "", err
	}
	defer resp.Body.Close()

	// write the file
	func() {
		out, err := os.Create(filepath.Join(d, version+".tar.gz"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}

		if _, err = io.Copy(out, resp.Body); err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
		out.Close()
	}()

	// calculate the sha
	e := exec.Command("sha256sum", filepath.Join(d, version+".tar.gz"))
	o, err := e.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return "", err
	}
	parts := strings.Split(string(o), " ")
	fmt.Println("new sha: " + parts[0])
	return parts[0], nil
}

type Input struct {
	Version string
	Sha     string
}

const formula = `# Copyright 2019 Google LLC
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
  url "https://github.com/GoogleContainerTools/kpt/archive/{{.Version}}.tar.gz"
  sha256 "{{.Sha}}"

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
