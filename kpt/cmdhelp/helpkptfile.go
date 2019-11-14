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

package cmdhelp

import "github.com/spf13/cobra"

var Kptfile = &cobra.Command{
	Use: "docs-kptfile",
	Long: `Description:
  A Kptfile resides at the root of each package or subpackage and contains
  package metadata.  Any time a package is fetched using kpt the Kptfile is
  updated with information about the source of the package, including the git
  commit.

  If the remote packaged does not have a Kptfile defined, then a local Kptfile
  will be created and populated with default values.

Schema:

	  apiVersion: kpt.dev/v1beta1
	  kind: Kptfile
	  metadata:
		name # the name of the package
	  packageMetadata:
		tags # tags for search and indexing.  Suggested tags: [app.kpt.dev/app-name]
		man # Kptfile relative path to man pages in md2man format.  Defaults to './MAN.md'
		url # url about the package
		email # email address of the package maintainer(s)
	  upstream:
		type: 'git'
		git:
		  repo # git repo url of the upstream package
		  commit # value of the remote commit that the package was copied from
		  directory # git subdirectory containing the package
		  ref # git ref that the commit was resolved from -- maybe a tag, branch, ref or commit
`,

	Example: `apiversion: kpt.dev/v1beta1
	kind: Kptfile
	metadata:
	  name: cockroachdb
	packageMetadata:
	  tags: ["app.kpt.dev/cockroachdb"]
	  url: https://example.com/package
	  email: maintainer@example.com
	upstream:
	  type: git
	    git:
	      repo: https://github.com/example/com
	      commit: dd7adeb5492cca4c24169cecee023dbe632e5167
	      directory: cockroachdb
	      ref: refs/heads/v1.0`,
}
