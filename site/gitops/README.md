# GitOps

Packages eventually have to be deployed into one or more clusters. The default
deployment mechanism is [Config Sync](gitops/configsync/), but since configuration is stored in standard
repositories, other [GitOps](https://opengitops.dev/) tools can also be used.

Currently supported gitops tools:
* [Config Sync](gitops/configsync/)