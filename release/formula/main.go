// Copyright 2019 The kpt Authors
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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(_ context.Context) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("must specify new version")
	}

	version := os.Args[1]
	url := "https://github.com/GoogleContainerTools/kpt/archive/" + version + ".tar.gz"

	formula, err := buildFormula(http.DefaultClient, url)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join("Formula", "kpt.rb"), []byte(formula), 0644)
	if err != nil {
		return err
	}
	return nil
}

func buildFormula(httpClient *http.Client, url string) (string, error) {
	sha256, err := hashURL(httpClient, url, sha256.New())
	if err != nil {
		return "", err
	}

	// generate the formula text
	formula := formulaTemplate
	formula = strings.ReplaceAll(formula, "{{url}}", url)
	formula = strings.ReplaceAll(formula, "{{sha256}}", sha256)

	return formula, nil
}

func hashURL(httpClient *http.Client, url string, hasher hash.Hash) (string, error) {
	fmt.Printf("fetching %q\n", url)

	// get the content
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("error getting %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected response from %q: %v", url, resp.Status)
	}

	if _, err := io.Copy(hasher, resp.Body); err != nil {
		return "", fmt.Errorf("error hashing response from %q: %w", url, err)
	}

	// calculate the sha
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

const formulaTemplate = `# Copyright 2019 The kpt Authors
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
  url "{{url}}"
  sha256 "{{sha256}}"

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
