# Copyright 2019, 2026 The kpt Authors
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
  desc "A toolchain for composing, customizing, and deploying Kubernetes packages at scale"
  homepage "https://kpt.dev"
  url "https://github.com/kptdev/kpt/archive/v1.0.0-beta.67.tar.gz"
  sha256 "358f6dc09e0ed494432d1b669ee1a9133bc20c285defbdc495ca29994b43b424"

  depends_on "go" => :build

  def install
    ENV["GO111MODULE"] = "on"
    system "go", "build", "-ldflags", "-X github.com/kptdev/kpt/run.version=#{version}", *std_go_args
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/kpt version")
  end
end
