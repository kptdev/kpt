In this section, we'll describe the typical workflows in kpt. We say "typical", because there is no
single right way of using kpt. A user may choose to use some command but not another. This
modularity is a key design principle. However, we still want to provide guidance on how the
functionality could be used in a end-to-end real-world scenarios.

A workflow in kpt can be best modelled as performing some verbs on the noun _package_.

For example, when consuming an upstream package, the initial workflow can look like this:

![img](/static/images/lifecycle/flow1.svg)

- **Get**: Using `kpt pkg get`
- **Explore**: Using an editor or running commands such as `kpt pkg tree`
- **Customize**: Manual edits or using `kpt fn eval`
- **Render**: Using `kpt fn render`

First, you get a package from upstream. Then, you explore the content of the package to understand
it better. Then you typically want to customize the package for you specific needs. Finally, you
render the package which produces the final resources that can be directly applied to the cluster.

This is an iterative process. There is usually a tight customize-render loop in order to produce
the desired outcome.

Some time later, you may want to update to a newer version of the upstream package:

![img](/static/images/lifecycle/flow2.svg)

- **Update**: Using `kpt pkg update`

Updating the package involves merging your local changes with the changes made by the upstream
package authors between the two specified versions.

Now, let's say you have rendered the package, and want to deploy it to a cluster. The workflow
may look like this:

![img](/static/images/lifecycle/flow3.svg)

- **Initialize**: One-time process using `kpt live init`
- **Preview**: Using `kpt live preview`
- **Apply**: Using `kpt live apply`
- **Observe**: Using `kpt live status`

First, you preview deployment of the package to the cluster. Then if the preview looks good,
you apply the package. Afterwards, you may observe the status of the package on the cluster.

You typically want to store the customized package on Git:

![img](/static/images/lifecycle/flow3.svg)

- **Publish**: Using `git commit`

The publishing flow is orthogonal to deployment flow. This allows you to act as a publisher of an
upstream package even though you may not deploy the package personally.
