## Frequent Asked Questions

Frequently Asked Questions

Q: **How does kpt fit in with the Kubernetes ecosystem?**
A: `kpt` is intended to be composed with other tools from the Kubernetes ecosystem.
   Rather than attempting to solve all problems related to configuration, kpt
   is focused solving how to publish and consume configuration packages.  `kpt`
   was developed to complement both the OSS Kubernetes project tools such as
   `kubectl` and `kustomize`, and other tools developed as part of the broader
   ecosystem.

Q: **How do I use kpt with the Kubernetes project tools?**
A: `kpt` may be used to publish, fetch and update configuration packaging.
   The project tools may be used to manipulate and apply the fetched configuration.

Q: **How can I use kpt to create blueprint packages?**
A: Blueprints may be published as `kpt` packages, using kpt for fetching and
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

Q: **What are setters?**
A: Setters are similar to the imperative `kubectl set` commands, but operate against local
   Resource configuration and are configured as annotations on Resource fields.
   See `kustomize help config set` for more information.

Q: **How do I parameterize kpt packages?**
A: For performing simple substitutions, `kustomize config set` may be used to replace
   marker values with values provided on the commandline.
   Alternatively, substitutions may be externalized from the package using configuration functions
   which can generate or transform configuration.
   See `kustomize help config set` and `kustomize help config run` for more information.

Q: **Does kpt work for non-Resource packages such as Terraform or Helm Charts?**
A: Sorta, kpt packages can contain non-Resource packaging artifacts.  These
   artifacts do not support Resource specific operations -- e.g.
   The `update` command `resource-merge` strategy will not work against them,
   but the `alpha-git-patch` strategy will.
   
Q: **How can I apply kpt packages to a cluster?**
A: kpt is designed to work with the OSS Kubernetes Kubernetes tools such
   as `kubectl` and `kustomize` -- e.g. `kustomize apply PKG/` or
   `kustomize config cat PKG/ | kubectl apply -f -`.
   Use `kustomize config cat` so that only non-config function Resources are applied.


