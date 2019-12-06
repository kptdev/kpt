## Frequent Asked Questions

Frequently Asked Questions

Q: **How does kpt fit in with other OSS Kubernetes tools?**
A: kpt is intended to be composed with other tools from the Kubernetes ecosystem.
   Rather than attempting to solve all problems related to configuration, kpt
   is focused solving how to publish and consume configuration packages.  kpt
   was developed to complement the OSS Kubernetes project tooling such as
   `kubectl` and `kustomize`.

Q: **How do I use kpt with the Kubernetes project tooling?**
A: kpt may be used to publish, fetch and update configuration packaging.
   The project tooling may be used to manipulate and apply the fetched configuration.

Q: **How can I use kpt to create blueprint packages?**
A: Blueprints may be published as kpt packages, using kpt for fetching and
   updating the packages from upstream.
   The local copy of the blueprint package may be directly edited, or other
   customization techniques may be applied (e.g. `kustomize build`).

Q: **What are some examples of blueprint customization techniques?**
A: - Using configuration functions (i.e. `kustomize config run`)
   - Using Kustomizations (i.e. `kustomize build`)
   - Duck-typed setter commands (i.e. `kustomize duck CMD`)

Q: **What are configuration functions?**
A: Configuration functions are programs applied to configuration which may generate, transform
   or validate new or existing configuration.  Functions are typically published as container
   images applied to a configuration package.
   See `kustomize help config run` for more information.

Q: **What are duck-typed setter commands?**
A: Duck-typed setter commands are similar to the imperative `kubectl set` commands,
   but operate against local Resource configuration and use duck-typing to identify
   the supported setters.
   See `kustomize help duck` for more information.

Q: **How do I parameterize kpt packages?**
A: Packages may be parameterized by using templating with configuration functions.
   The function encapsulates the parameterized version of theResource, and emits
   the expanded Resource.
   See `kustomize help config run` for more information.

Q: **Does kpt work for non-Resource packages such as Terraform or Helm Charts?**
A: Yes, kpt packages can contain non-Resource packaging artifacts.  These
   artifacts do not support Resource specific operations -- e.g.
   The `update` command `resource-merge` strategy will not work against them,
   but the `alpha-git-patch` strategy will.
   
Q: **How can I apply kpt packages to a cluster?**
A: kpt is designed to work with the OSS Kubernetes Kubernetes tools such
   as `kubectl` and `kustomize` -- e.g. `kustomize config cat PKG/ | kubectl apply -f -`.
   Use `kustomize config cat` so that only non-config function Resources are applied.
   

