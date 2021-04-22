### Overview

#### What is kpt?

> A Git-native, semantic-aware, extensible client-side tool for 
> packaging, customizing, validating, and applying Kubernetes resources.

#### Problem space

kpt is targeted at configuration and policy enforcement of kubernetes 
infrastructure at scale.  kpt design is aimed to solve the 
following problems:

1. Quick creation of configuration blueprints from existing YAML files 
without introducing a template or domain specific language into your infrastructure stack.
1. Updating downstream forks of configuration packages with upstream 
changes with minimal effort and repetitive work.
1. A command line (imperative) and configuration file (declarative) way to 
customize the configuration and enforce policy.
1. Packaging up parts of customization or validation pipeline into 
reusable building blocks (KRM functions) and sharing them within your 
enteprise.
1. Applying a group of resources as a package which alleviates 
the need to use namespaces or labels for pruning.

#### Tool interoperability

kpt works well with other tools that rely on KRM as the data format:

1. You can easily replace remote bases in kustomize with kpt packages and 
avoid network dependency.
1. kpt works well with [ConfigSync] which keeps your cluster in sync with 
a git repo.
1. You can pipe KRM based configuration to kpt, process or apply those or 
pipe out the results to downstream tools.
1. kpt supports editing files locally using your favorite editor.

#### Next steps

If you'd like to go through a step by step introduction to kpt and it's 
concepts, the best way to get started is to read the [Kpt Book]


----
[ConfigSync]: https://cloud.google.com/kubernetes-engine/docs/add-on/config-sync/overview
[Kpt Book]: /book/