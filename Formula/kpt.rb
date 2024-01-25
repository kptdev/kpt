# Copyright 2019 The kpt Authors
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
  url "https://github.com/GoogleContainerTools/kpt/archive/v1.0.0-beta.49.tar.gz"
  sha256 "e8f5beb1b4538f0b5d5322cb1600b93728e4a90a975860873883507ee144a262"

  depends_on "go" => :build

  def install
    ENV["GO111MODULE"] = "on"
    system "go", "build", "-ldflags", "-X github.com/GoogleContainerTools/kpt/run.version=#{version}", *std_go_args
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/kpt version")
  end
end
