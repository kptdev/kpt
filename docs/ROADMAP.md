# kpt roadmap for 2021

Last updated: June 1st, 2021

Draft of the *v1* release slated for Summer 2021.

### 1. Declarative function pipeline

kpt has added a declarative way to customize and validate configuration.  
This allows you to run several mutation and validation 
functions in a pipeline alleviating the need to create shell scripts that do 
the same thing.  Further information can be founnd in the 
[declarative function execution]  section of the Kpt Book.

### 2. Setters

Setters used to be a special entity without sufficient differentiation from
KRM functions. In kpt v1 setters become just another function and drastically
simplify syntax.  Configuring 4 setters used to take 20 lines of yaml 
and now takes 6.  Setters also get recursively applied to sub-packages by
default.  For further information on the setter function please visit: 
[apply-setters documentation]. 

### 3. Resource merge

kpt package updates now default to the resource merge strategy 
which allows you to edit configuration with an text editor of your choice 
and still be able to rebase with upstream changes. 

### 4. Live apply

_kpt live_ used to use ConfigMap to store inventory information. This was
convenient as it didn't require any CRDs, but it had challenges around encoding
of the GroupKind, name and namespace, and it didn't allow us to easily add
additional metadata about a package, such as the package version. This is
is now migrated to use ResourceGroup CRD.  You can learn more about in the
[apply chapter] of kpt Book.
https://kpt.dev/book/06-apply/

The majority of the code for kpt live is in the [cli-utils] repo. Our
goal is to keep this code independent of kpt and provide the apply logic as a
library that can be used by other tools. The API is still going through some
changes, but we are actively working to stabilize it.

### 5. Updated documentation

Documentation was a major area of investment, including the [kpt Book].
The book is a methodical way to introduce some unique kpt concepts like 
in place editing and hydration.  It's meant to be a hands on guide where the user
configures and deploys wordpress and nginx while learning about the kpt
concepts.

### 6. Function catalog

All of the hydration and validation logic has been moved from the kpt binary 
to functions allowing for flexibility and security.  This enables new 
scenarios like limiting the customization and validation to a subset of 
allowed functions.  The function catalog has received additional functions, 
examples and help. Please visit the [function catalog] for further information.

## Ongoing work
Since this is a draft of the release notes you should be aware of the
ongoing work.  At the time of writing the team is in [Milestone 3], and we
are planning to complete the work in [Milestone 4].  For instance as of 6/1/21
[out of place hydration] support is still in the works.  The exact dates
and features are going to be highly dependent on feedback we get from beta
testers.

## Feedback channels:
kpt-users@googlegroups.com
File a [new issue] on Github, but please search first. 


[new issue]: https://github.com/GoogleContainerTools/kpt/issues/new/choose
[declarative function execution]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution
[apply-setters documentation]: https://catalog.kpt.dev/apply-setters/v0.1/ 
[kpt Book]: https://kpt.dev/book/
[apply chapter]: https://kpt.dev/book/06-apply/
[cli-utils]: https://github.com/kubernetes-sigs/cli-utils
[function catalog]: https://catalog.kpt.dev/
[Milestone 3]: https://github.com/GoogleContainerTools/kpt/milestone/10
[Milestone 4]: https://github.com/GoogleContainerTools/kpt/milestone/11
[out of place hydration]: https://github.com/GoogleContainerTools/kpt/issues/1412