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

var PackageStructure = &cobra.Command{
	Use: "docs-package-structure",
	Long: `Description:
  kpt packages may be published as:

    * git repositories
    * git repository subdirectories

  kpt packages are packages of Resource Configuration as yaml files.
  As such they SHOULD contain at least one of:
  
    * Kubernetes Resource Configuration files (.yaml or .yml)
    * Kustomization.yaml
    * Subdirectories containing either of the above

 kpt packages MAY additionally contain:

    * Kptfile: package metadata (see 'kpt help kptfile')
    * MAN.md: package documentation (md2man format)
    * LICENSE: package LICENSE
    * Other kpt subpackages
    * Arbitrary files

  A configuration directory may be blessed with recommended kpt package metadata
  files using 'kpt bless dir/'
 `,
	Example: ` # * 1 resource per-file
  # * flat structure
  
  $ tree cockroachdb
  cockroachdb/
  ├── Kptfile
  ├── MAN.md
  ├── cockroachdb-pod-disruption-budget.yaml
  ├── cockroachdb-public-service.yaml
  ├── cockroachdb-service.yaml
  └── cockroachdb-statefulset.yaml

  # * multiple resources per-file
  # * nested structure
  # * contains subpackage

  $ tree wordpress/
  wordpress/
  ├── Kptfile
  ├── MAN.md
  ├── Kustomization.yaml
  ├── mysql
  │   ├── Kptfile
  │   └── mysql.yaml
  └── wordpress
      └── wordpress.yaml
`,
}
