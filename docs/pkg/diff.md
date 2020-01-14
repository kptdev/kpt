## diff

Show changes between local and upstream source package

### Synopsis

    kpt pkg diff [LOCAL_PKG_DIR@VERSION]

Diff commands lets you answer the following questions:
  - What have I changed in my package relative to the upstream source package
  - What has changed in the upstream source package between the original source version and target version

You can specify a diffing tool with options to show the changes. By default, it
uses 'diff' commandline tool.

Args:

  LOCAL_PKG_DIR:
    Local package to compare. Command will fail if the directory doesn't exist, or does not
    contain a Kptfile.  Defaults to the current working directory.

  VERSION:
    A git tag, branch, ref or commit. Specified after the local_package with @ -- pkg_dir@version.
    Defaults to the local package version that was last fetched.

Envs:

  KPT_EXTERNAL_DIFF:
   Commandline diffing tool (diff by default) that will be used to show changes. For ex.
   # Use meld to show changes
   KPT_EXTERNAL_DIFF=meld kpt pkg diff

  KPT_EXTERNAL_DIFF_OPTS:
   Commandline options to use for the diffing tool. For ex.
   # Using "-a" diff option
   KPT_EXTERNAL_DIFF_OPTS="-a" kpt pkg diff --diff-tool meld

Flags:
  diff-type:
    The type of changes to view (local by default). Following types are supported:
	 local: shows changes in local package relative to upstream source package at original version
	 remote: shows changes in upstream source package at target version relative to original version
	 combined: shows changes in local package relative to upstream source package at target version
	 3way: shows changes in local package and source package at target version relative to original version side by side

  diff-tool:
    Commandline tool (diff by default) for showing the changes.
	# Show changes using 'meld' commandline tool
	kpt pkg diff @master --diff-tool meld

	Note that it overrides the KPT_EXTERNAL_DIFF environment variable.

  diff-opts:
    Commandline options to use with the diffing tool.
	# Show changes using "diff" with recurive options
	kpt pkg diff @master --diff-tool meld --diff-opts "-r"

	Note that it overrides the KPT_EXTERNAL_DIFF_OPTS environment variable.

### Examples

    # Show changes in current package relative to upstream source package
    kpt pkg diff

    # Show changes in current package relative to upstream source package using meld tool with auto compare option.
    kpt pkg diff --diff-tool meld --diff-tool-opts "-a"

    # Show changes in upstream source package between current version and target version
    kpt pkg diff @v4.0.0 --diff-type remote

    # Show changes in current package relative to target version
    kpt pkg diff @v4.0.0 --diff-type combined

    # Show 3way changes between the local package, upstream package at original version and upstream package at target version using meld
    kpt pkg diff @v4.0.0 --diff-type 3way --diff-tool meld --diff-tool-opts "-a"