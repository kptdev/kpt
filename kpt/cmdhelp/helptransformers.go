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

var Reconcilers = &cobra.Command{
	Use: "reconcilers",
	Long: `Description:
  Reconcilers are client-side versions of the Kubernetes Controller Reconcile functions which
  implement Kubernetes APIs in the cluster.

  ### Reconcilers may be used to:

  - Generate new Resources from abstractions -- may use templates
  - Apply cross-cutting values to all Resource in the package
  - Enforce cross-cutting policy constraints amongst all Resource in the package
  - Enforce cross-cutting linting constraints amongst all Resources in the package

  ### Examples of Reconcilers:

  - Replace a field on some / all Resources from the reconciler config or an environment variable
  - Define abstractions and generate Resources using templates, DSLs, typescript programs, etc
  - Validate all container images use a tag
  - Validate all workloads have a PodDisruptionBudget

  ### Details

  Reconcilers are configured as Resources within the package, and are identified by one of:

  - having an apiVersion starting with ` + "`" + `*.gcr.io` + "`" + ` or ` + "`" + `docker.io` + "`" + `
  - having an annotation ` + "`" + `kpt.dev/container: CONTAINER_NAME` + "`" + `

  Example Reconciler config in pkg/example-reconciler.yaml which would be triggered
  by running ` + "`" + `kpt reconcile pkg/` + "`" + `

	apiVersion: gcr.io/kpt-dev/example-reconciler
	kind: ExampleReconciler
	spec:
	  someField: someValue

  In the preceding example, ` + "`" + `kpt reconcile pkg/` + "`" + ` would:

  - run a new container from the image gcr.io/kpt-dev/example-reconciler
  - provide the current contents of pkg/ package to the container on stdin
  - read the new package contents from the container on stdout
  - copy container stderr to the console stderr
  - fail if the container exists non-0

  ### Reconciler input format:

  Reconcilers take a ResourceList input that contains:

  1. *items*: the list of Resources in package -- these are the items to be modified
  2. *functionConfig*: the configuration for how to modify the items -- this is the
     Resource with *apiVersion: gcr.io/...* which triggered the Reconcile.

	apiVersion: kpt.dev/v1alpha1
	kind: ResourceList
	functionConfig:
	  apiVersion: example.com/v1alpha1
	  kind: ExampleReconciler
	  spec:
	    config: value
	items: # input values read from the reconciled package
	- apiVersion: apps/v1
	  kind: Deployment
	  spec:
	    template: {}
	- apiVersion: v1
	  kind: Service
    
  ### Reconciler output format:

  Reconcilers emit a ResourceList output that contains the updated items.  It may optionally
  include the functionConfig (ignored).

	apiVersion: kpt.dev/v1alpha1
	kind: ResourceList
	items: # output values updated by the Reconciler
	- apiVersion: apps/v1
	  kind: Deployment
	  spec:
	    template: {} # updated values
	- apiVersion: v1
	  kind: Service

  The Reconciler container image is run with the following options:
  > [--rm, -i, -a=STDIN, -a=STDOUT, -a=STDERR, --network=none, --user=nobody, --security-opt=no-new-privileges]

  ### Writing Reconcilers:

  #### Manually running Reconciler programs outside of a container -- for development

  When developing a Reconciler, it may be desirable to manually run the Reconciler outside of a container.
  This is possible using "kpt cat" to generate the reconciler input for a package and piping it to
  the Reconciler program.

` + "  - Manually generate input through: `kpt cat --function-config config.yaml --wrap-kind ResourceList --wrap-version kpt.dev/v1alpha1" + `
    - wraps the pkg/ and config.yaml in a *ResourceList* and writes to stdout
` + "    - e.g. `kpt cat --function-config config.yaml --wrap-kind ResourceList --wrap-version kpt.dev/v1alpha1 | ./reconciler`" + `
` + "  - Manually unwrap the output through: `kpt cat`" + `
  - unwraps the output from the ResourceList produced by the Reconciler
` + "    - e.g. `... | ./reconciler | kpt cat`" + `

  #### Simplifying Reconciler implementation

  The Reconciler interface reads the full list of package resources + the function-config from stdin
  and emits the modified package contents on stdout.  Parsing and modifying the inputs may be
  non-trivial for simple Reconcilers -- such as templates.

  kpt offers several commands that can be combined using pipes that can parse and update Resources.

` + "  - `kpt reconcile xargs` parses the functionConfig into flags and environment variables and invokes the Reconciler" + `
` + "    - e.g. `kpt reconcile xargs -- ./reconciler`" + `
  
` + "  - `kpt merge` merges Resources specified multiple times, allowing templates to simply emit new copies of Resources which be merged into the local package copy." + `
` + "    - e.g. `... ./reconciler | kpt merge`" + `

` + "  - `kpt fmt` formats Resources and generates file names for newly created Resources." + `
` + "    - e.g. `... ./reconciler | kpt fmt --set-filenames`" + `
` + "    - **note**: `kpt merge` must come before `kpt fmt` to prevent filenames from being overridden" + `

` + "  - `kpt reconcile wrap` combines `xargs`, `merge`, `fmt` into a single command and is the recommended approach for templating abstractions." + `
` + "    - e.g. `kpt reconcile wrap -- ./reconciler`" + `

  ### Full examples:

  - https://github.com/GoogleContainerTools/kpt/testutil/cockroachdb-simple
  - https://github.com/GoogleContainerTools/kpt/testutil/cockroachdb-reconciler
  - https://github.com/GoogleContainerTools/kpt/testutil/resource-policy-reconciler

  ## Local e2e example for running locally outside of ` + "`" + `kpt reconcile` + "`" + `

	$ kpt cat pkg/ --function-config pkg/config.yaml --wrap-kind ResourceList | \
	kpt reconcile wrap -- ./reconciler | kpt cat

  ## Simple template Reconciler container

  > reconciler/Dockerfile

	FROM golang:1.13-stretch
	
	# build kpt from local source
	# TODO(pwittrock): fetch the binary when there is a release
	WORKDIR /go/src/
	COPY lib lib/
	COPY cmd/ cmd/
	WORKDIR /go/src/cmd
	RUN go build -v -o /bin/kpt .
	
	COPY template.sh /usr/bin/reconciler
	CMD ["reconciler"]
	
  > reconciler/template.sh
	
	#!/bin/bash
	if [ -z ${KPT_WRAPPED} ]; then
	  export KPT_WRAPPED=true
	  kpt reconcile wrap -- $0
	  exit $?
	fi
	
	cat <<End-of-message
	kind: Service
	metadata:
	  name: ${NAME}
	spec:
	  ports:
	  - port: ${PORT}
		protocol: TCP
		targetPort: ${PORT}
	  selector:
		run: ${NAME}
	---
	apiVersion: apps/v1
	kind: Deployment
	metadata:
	  labels:
		run: ${NAME}
	  name: ${NAME}
	spec:
	  selector:
		matchLabels:
		  run: ${NAME}
	  template:
		metadata:
		  labels:
			app: ${NAME}
		spec:
		  containers:
		  - name: ${NAME}
			image: ${IMAGE}
			ports:
			- containerPort: ${PORT}
	
  ## Resource as part of package picked up by 'kpt reconcile pkg/'

  > pkg/application.yaml

	apiVersion: gcr.io/myproject/myimage:imageversion
	kind: Application
	metadata:
	  name: sample-name
	spec:
	  image: gcr.io/sample/image:version
	  port: 8080
`,
}
