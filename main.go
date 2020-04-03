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

//go:generate $GOBIN/mdtogo site/content/en/reference/live internal/docs/generated/livedocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/content/en/reference/pkg internal/docs/generated/pkgdocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/content/en/reference/cfg internal/docs/generated/cfgdocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/content/en/reference/fn internal/docs/generated/fndocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/content/en/reference internal/docs/generated/overview --license=none --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/content/en/guides/consumer internal/guides/generated/consumer --license=none --recursive=true --strategy=guide
//go:generate $GOBIN/mdtogo site/content/en/guides/ecosystem internal/guides/generated/ecosystem --license=none --recursive=true --strategy=guide
//go:generate $GOBIN/mdtogo site/content/en/guides/producer internal/guides/generated/producer --license=none --recursive=true --strategy=guide
package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt/run"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd := run.GetMain()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
