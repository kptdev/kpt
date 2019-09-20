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

package cmdtutorials

import "github.com/spf13/cobra"

var Tutorials = &cobra.Command{
	Use:     "tutorials",
	Short:   `Contains tutorials for using kpt`,
	Long:    `Contains tutorials for using kpt.`,
	Example: `kpt tutorials 1-fetch-a-package`,
}

func init() {

	Tutorials.AddCommand(
		&cobra.Command{
			Use:   "1-fetch-a-package",
			Short: "Tutorial for fetching a package from a remote source",
			Long: `Packages are directories of Kubernetes Resource Configuration which
may be fetched from sources such as git:

- No additional package metadata or structure is required
- Natively compatible with 'kubectl apply'
- May be fetched and updated to specific revisions (using git tags or branches).
- May contain non-configuration files or metadata as part of the package

## Fetch a remote package

### Fetching cassandra

  Fetch a "raw" package (e.g. config only -- no kpt metadata) from the kubernetes examples repo.

	kpt get  https://github.com/kubernetes/examples/cassandra cassandra/

  'kpt get' fetched the remote package from HEAD of the
  https://github.com/kubernetes/examples master branch.

	$ kpt tree cassandra/
	cassandra
	├── [cassandra-service.yaml]  v1.Service cassandra
	├── [cassandra-statefulset.yaml]  apps/v1.StatefulSet cassandra
	└── [cassandra-statefulset.yaml]  storage.k8s.io/v1.StorageClass fast
	
  'kpt tree' printed the package structure -- displaying both the Resources as well as the
  files the Resources are specified in.

	$ kpt desc cassandra
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+
	| LOCAL DIRECTORY |   NAME    |           SOURCE REPOSITORY            |  SUBPATH  | VERSION | COMMIT  |
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+
	| cassandra       | cassandra | https://github.com/kubernetes/examples | cassandra | master  | 1543966 |
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+

  'kpt desc LOCAL_PACKAGE' prints information about the source of the package -- e.g. 
  the repo, subdirectory, etc.

### Fetch the guestbook package

	$ kpt get https://github.com/kubernetes/examples/guestbook ./my-guestbook-copy

  The guestbook package contains multiple guest book instances in separate
  subdirectories.

	$ kpt tree my-guestbook-copy/
	my-guestbook-copy
	├── [frontend-deployment.yaml]  apps/v1.Deployment frontend
	├── [frontend-service.yaml]  v1.Service frontend
	├── [redis-master-deployment.yaml]  apps/v1.Deployment redis-master
	├── [redis-master-service.yaml]  v1.Service redis-master
	├── [redis-slave-deployment.yaml]  apps/v1.Deployment redis-slave
	├── [redis-slave-service.yaml]  v1.Service redis-slave
	├── all-in-one
	│   ├── [frontend.yaml]  apps/v1.Deployment frontend
	│   ├── [frontend.yaml]  v1.Service frontend
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment frontend
	│   ├── [guestbook-all-in-one.yaml]  v1.Service frontend
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-master
	│   ├── [guestbook-all-in-one.yaml]  v1.Service redis-master
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-slave
	│   ├── [guestbook-all-in-one.yaml]  v1.Service redis-slave
	│   ├── [redis-slave.yaml]  apps/v1.Deployment redis-slave
	│   └── [redis-slave.yaml]  v1.Service redis-slave
	└── legacy
		├── [frontend-controller.yaml]  v1.ReplicationController frontend
		├── [redis-master-controller.yaml]  v1.ReplicationController redis-master
		└── [redis-slave-controller.yaml]  v1.ReplicationController redis-slave

  The separate guestbook subpackages contain variants of the same guestbook application.
  To fetch only the all-in-one instance, specify that subdirectory as part of the package.

	$ kpt get https://github.com/kubernetes/examples/guestbook/all-in-one ./new-guestbook-copy

  'kpt get' only fetched the all-in-one subpackage.

	$ kpt tree new-guestbook-copy
	new-guestbook-copy
	├── [frontend.yaml]  apps/v1.Deployment frontend
	├── [frontend.yaml]  v1.Service frontend
	├── [guestbook-all-in-one.yaml]  apps/v1.Deployment frontend
	├── [guestbook-all-in-one.yaml]  v1.Service frontend
	├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-master
	├── [guestbook-all-in-one.yaml]  v1.Service redis-master
	├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-slave
	├── [guestbook-all-in-one.yaml]  v1.Service redis-slave
	├── [redis-slave.yaml]  apps/v1.Deployment redis-slave
	└── [redis-slave.yaml]  v1.Service redis-slave

## Fetch a specific version of a package

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0 cockroachdb/

  Specifying '@version' after the package uri fetched the package at that revision.
  The version may be a git branch, tag or ref.

## Import a package from a Helm chart

	helm fetch stable/redis
	helm template redis-9.* | kpt get - ./redis-9/

  This imported the expanded package Resources from stdin and created a local kpt package.

	$ kpt tree redis-9/
	redis-9
	├── [release-name-redis-headless_service.yaml]  v1.Service release-name-redis-headless
	├── [release-name-redis-health_configmap.yaml]  v1.ConfigMap release-name-redis-health
	├── [release-name-redis-master_service.yaml]  v1.Service release-name-redis-master
	├── [release-name-redis-master_statefulset.yaml]  apps/v1beta2.StatefulSet release-name-redis-master
	├── [release-name-redis-slave_service.yaml]  v1.Service release-name-redis-slave
	├── [release-name-redis-slave_statefulset.yaml]  apps/v1beta2.StatefulSet release-name-redis-slave
	├── [release-name-redis_configmap.yaml]  v1.ConfigMap release-name-redis
	└── [release-name-redis_secret.yaml]  v1.Secret release-name-redis

  The names of the Resource files may be configured using the --pattern flag.

	helm fetch stable/redis
	helm template redis-9.* | kpt get - ./redis-9/ --pattern '%n.resource.yaml'
	
  This configured the generated resource file names to be RESOURCENAME.resource.yaml
  instead of RESOURCENAME_RESOURCETYPE.yaml
  Multiple Resources with the same name are put into the same file:

	$ kpt tree redis-9/
	redis-9
	├── [release-name-redis-headless.resource.yaml]  v1.Service release-name-redis-headless
	├── [release-name-redis-health.resource.yaml]  v1.ConfigMap release-name-redis-health
	├── [release-name-redis-master.resource.yaml]  v1.Service release-name-redis-master
	├── [release-name-redis-master.resource.yaml]  apps/v1beta2.StatefulSet release-name-redis-master
	├── [release-name-redis-slave.resource.yaml]  v1.Service release-name-redis-slave
	├── [release-name-redis-slave.resource.yaml]  apps/v1beta2.StatefulSet release-name-redis-slave
	├── [release-name-redis.resource.yaml]  v1.ConfigMap release-name-redis
	└── [release-name-redis.resource.yaml]  v1.Secret release-name-redis
	
 Run 'kpt help get' for the set of --pattern options

## Expand Kustomized Configuration into a separate package

  Kustomization directories are natively recognized as kpt packages, however they may
  also be expanded into separate packages.

	kpt get https://github.com/kubernetes-sigs/kustomize/examples/wordpress wordpress/
	kustomize build wordpress/ | kpt get - wordpress-expanded/

  This expanded the Kustomization into a new package

	$ kpt tree wordpress-expanded/
	wordpress-expanded
	├── [demo-mysql-pass_secret.yaml]  v1.Secret demo-mysql-pass
	├── [demo-mysql_deployment.yaml]  apps/v1beta2.Deployment demo-mysql
	├── [demo-mysql_service.yaml]  v1.Service demo-mysql
	├── [demo-wordpress_deployment.yaml]  apps/v1beta2.Deployment demo-wordpress
	└── [demo-wordpress_service.yaml]  v1.Service demo-wordpress
`,
		})

	Tutorials.AddCommand(
		&cobra.Command{
			Use:   "2-working-with-local-packages",
			Short: "Tutorial for  with local packages",
			Long: `Kpt provides various tools for working with local packages once they are fetched.

  First stage a package to work with

	kpt get  https://github.com/kubernetes/examples/mysql-wordpress-pd wordpress/

## Viewing package structure

	$ kpt tree wordpress
	wordpress
	├── [gce-volumes.yaml]  v1.PersistentVolume wordpress-pv-1
	├── [gce-volumes.yaml]  v1.PersistentVolume wordpress-pv-2
	├── [local-volumes.yaml]  v1.PersistentVolume local-pv-1
	├── [local-volumes.yaml]  v1.PersistentVolume local-pv-2
	├── [mysql-deployment.yaml]  v1.PersistentVolumeClaim mysql-pv-claim
	├── [mysql-deployment.yaml]  apps/v1.Deployment wordpress-mysql
	├── [mysql-deployment.yaml]  v1.Service wordpress-mysql
	├── [wordpress-deployment.yaml]  apps/v1.Deployment wordpress
	├── [wordpress-deployment.yaml]  v1.Service wordpress
	└── [wordpress-deployment.yaml]  v1.PersistentVolumeClaim wp-pv-claim

  tree summarizes the package Files and Resources

## View the package Resources

	$ kpt cat wordpress/
	apiVersion: v1
	kind: PersistentVolume
	metadata:
	  name: wordpress-pv-1
	  annotations:
		io.kpt.dev/mode: 420
		io.kpt.dev/package: .
		io.kpt.dev/path: gce-volumes.yaml
	spec:
	  accessModes:
	  - ReadWriteOnce
	  capacity:
		storage: 20Gi
	  gcePersistentDisk:
		fsType: ext4
		pdName: wordpress-1
	---
	apiVersion: v1
	...

  cat prints the raw package Resources.

## Format the Resources for a package (like go fmt)

	$ kpt fmt wordpress/

  fmt formats the Resource Configuration by applying a consistent ordering of fields
  and indentation.

## Search for local package Resources by field

	$ kpt grep "metadata.name=wordpress" wordpress/
	apiVersion: v1
	kind: Service
	metadata:
	  name: wordpress
	  labels:
		app: wordpress
	  annotations:
		io.kpt.dev/mode: 420
		io.kpt.dev/package: .
		io.kpt.dev/path: wordpress-deployment.yaml
	spec:
	  ports:
	  - port: 80
	  selector:
		app: wordpress
		tier: frontend
	  type: LoadBalancer
	---
	...

  grep prints Resources matching some field value.  The Resources are annotated with their
  file source so they can be piped to other commands without losing this information.

	$ kpt grep "spec.status.spec.containers[name=nginx].image=mysql:5\.6" wordpress/
	apiVersion: apps/v1 # for k8s versions before 1.9.0 use apps/v1beta2  and before 1.8.0 use extensions/v1beta1
	kind: Deployment
	metadata:
	  name: wordpress-mysql
	  labels:
		app: wordpress
	spec:
	  selector:
		matchLabels:
		  app: wordpress
		  tier: mysql
	  template:
		metadata:
		  labels:
			app: wordpress
			tier: mysql
	...

  - list elements may be indexed by a field value using list[field=value]
  - '.' as part of a key or value may be escaped as '\.'

## Combine grep and tree

	$ kpt grep "metadata.name=wordpress" wordpress/ | kpt tree
	.
	├── [wordpress-deployment.yaml]  apps/v1.Deployment wordpress
	└── [wordpress-deployment.yaml]  v1.Service wordpress

  tree will read from stdin if no arguments are provided.  grep can be used with
  tree to only print a subset of the package.

## Combing grep and get

	$ kpt grep "metadata.name=wordpress" wordpress/ | kpt get - ./new-wordpress

  get will create a new package from the Resource Config emitted by grep

	$ kpt tree new-wordpress/
	new-wordpress
	├── [wordpress_deployment.yaml]  apps/v1.Deployment wordpress
	└── [wordpress_service.yaml]  v1.Service wordpress

## Combine cat and get

	$ kpt cat pkg/ | my-custom-transformer | kpt get - pkg/

'cat' may be used with 'get' to perform transformations with unit pipes
`,
		})

	Tutorials.AddCommand(
		&cobra.Command{
			Use:   "3-update-a-local-package",
			Short: "Update a previously fetched package ",
			Long: `Local packages may be updated with upstream package changes.

- Updates may be merged using different strategies
  - Run 'kpt help update' for the list of supported update strategies
- If no new revision is specified in the update, and the source was a branch, then the package
  will be updated to the tip of that branch.
- The local package must be committed to git be updated 
- Updates to packages generated from stdin are not yet supported

## Update an unchanged package

  Stage the package to be updated

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0 cockroachdb/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Diff a local package vs a new upstream version

  NOTE: the diff viewer can be controlled by setting KPT_EXTERNAL_DIFF --
  'export KPT_EXTERNAL_DIFF=my-differ'.
  See 'kpt help diff' for more options.

	kpt diff cockroachdb/@v1.4 --diff-type remote
	diff ...
	8a9
	>     foo: bar
	67c68
	<   minAvailable: 67%
	---
	>   minAvailable: 70%
	77c78
	<   replicas: 3
	---
	>   replicas: 7

  Update the package to the new version.  This requires that the package is unmodified from when
  it was fetched.

	kpt update cockroachdb@v1.4
	git diff cockroachdb/

  The updates are unstaged and must be committed.

## Updating merging remote changes with local changes

  Stage the package to be updated

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0 cockroachdb/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Make local edits to the package

	sed -i '' 's/port: 8080/port: 8081/g' ./cockroachdb/cockroachdb-statefulset.yaml
	git add . && git commit -m 'change cockroachdb port from 8080 to 8081'

  Diff the local package vs the original source upstream package -- see what you've changed

	$ kpt diff cockroachdb/
	diff ...
	17c17
	<   - port: 8081
	---
	>   - port: 8080
	50c50
	<   - port: 8081
	---
	>   - port: 8080

  Diff the local package vs a new upstream version -- see what you will be updating to

	$ kpt diff cockroachdb/@v1.4 --diff-type combined
	diff ...
	>     foo: bar
	17c18
	<   - port: 8081
	---
	>   - port: 8080
	50c51
	<   - port: 8081
	---
	>   - port: 8080
	67c68
	<   minAvailable: 67%
	---
	>   minAvailable: 70%
	77c78
	<   replicas: 3
	---
	>   replicas: 7

  Update the package to a new version.

  **NOTE:** --strategy is required when the local package has been changed from its source.
  In this case we have changed the local port field, so we must specify a strategy.

	kpt update cockroachdb@v1.4 --strategy alpha-git-patch
	git diff HEAD^ HEAD

  This merged the upstream changes into the local package, and created a new git commit.

## Update with local merge conflicts

  Stage the package to be updated

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0 cockroachdb/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Make local edits to the package.  Edit a field that will be changed upstream.

	sed -i '' 's/replicas: 3/replicas: 11/g' ./cockroachdb/cockroachdb-statefulset.yaml
	git add . && git commit -m 'change cockroachdb replicas from 3 to 11'

  View the 3way diff -- requires a diff viewer capable of 3way diffs (e.g. meld)

	kpt diff cockroachdb/@v1.4 --diff-type 3way

  This will show that the replicas field cannot be merged without a conflict -- it has
  been changed both in the upstream new package version, and in the local package.

  Go ahead and update the package to a new version anyway.  Expect a merge conflict.

	kpt update cockroachdb@v1.4 --strategy alpha-git-patch

  View the conflict

	$ git diff
	++<<<<<<< HEAD
	 +  replicas: 11
	++=======
	+   replicas: 7
	++>>>>>>> update cockroachdb (https://github.com/pwittrock/examples) from v1.0 (1f356407c2bcd5c56907d366161cbca833679ed1) to v1.4 (a3ea1604962746cd157769ef305951bdd88c628a)

  Fix the conflict and continue with the merge
	
	nano -w ./cockroachdb/cockroachdb-statefulset.yaml
	git add  cockroachdb/
  	git am --continue

  View the updates:

	git diff HEAD^ HEAD

## Manually update by generating a patch

  Stage the package to be updated

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0 cockroachdb/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Update the package to a new version.  Expect a merge conflict.

	kpt update cockroachdb@v1.4 --strategy alpha-git-patch --dry-run > patch
	git am -3 --directory cockroachdb < patch

## Update to HEAD of the branch the package was fetched from

  Fetch the package

	kpt get https://github.com/your/repo/here@master here/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Make upstream changes to the package at https://github.com/your/repo/here on
  the master branch.  Then update it.

	kpt update here/

  This fetched the updates from the upstream master branch.
`,
		})

	Tutorials.AddCommand(
		&cobra.Command{
			Use:   "4-publish-a-package",
			Short: "Publish a new package",
			Long: `While packages may be published as directories of raw Configuration,
kpt supports blessing a directory with additional package metadata that can benift
package discovery.

	kpt bless my-package/ --name my-package --description 'fun new package'
	git add my-package && git commit -m 'new kpt package'
	git push origin master

  This blessed the package by creating a Kptfile and MAN.md.  The MAN.md may be
  modified to include package documentation, which can be displayed with 'kpt man local-copy/'
`,
		})
}
