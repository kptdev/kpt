## tutorials 3-update-a-local-package

Update a previously fetched package 

### Synopsis

Local packages may be updated with upstream package changes.

- Updates may be merged using different strategies
  - Run 'kpt help update' for the list of supported update strategies
- If no new revision is specified in the update, and the source was a branch, then the package
  will be updated to the tip of that branch.
- The local package must be committed to git be updated 
- Updates to packages generated from stdin are not yet supported

## Update an unchanged package

  Stage the package to be updated

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 cockroachdb/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Diff a local package vs a new upstream version

  NOTE: the diff viewer can be controlled by setting KPT_EXTERNAL_DIFF --
  'export KPT_EXTERNAL_DIFF=my-differ'.
  See 'kpt help diff' for more options.

	kpt diff cockroachdb/@v1.1.0 --diff-type remote
	diff ...
	76c76
	<   replicas: 3
	---
	>   replicas: 5
	118c118
	<         image: cockroachdb/cockroach:v1.1.0
	---
	>         image: cockroachdb/cockroach:v1.1.1


  Update the package to the new version.  This requires that the package is unmodified from when
  it was fetched.

	kpt update cockroachdb@v1.1.0
	git diff cockroachdb/

  The updates are unstaged and must be committed.

## Updating merging remote changes with local changes

  Stage the package to be updated

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 cockroachdb/
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

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 cockroachdb/
	git add cockroachdb/ && git commit -m 'fetch cockroachdb'

  Make local edits to the package.  Edit a field that will be changed upstream.

	kpt cockroachdb set replicas cockroachdb --value 11
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

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 cockroachdb/
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


```
tutorials 3-update-a-local-package [flags]
```

### Options

```
  -h, --help   help for 3-update-a-local-package
```

### SEE ALSO

* [tutorials](tutorials.md)	 - Contains tutorials for using kpt

