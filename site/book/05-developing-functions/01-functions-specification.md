In order to enable functions to be developed in different toolchains and
languages and be interoperable and backwards compatible, the kpt project created
a standard for the inter-process communication between the orchestrator (i.e.
kpt CLI) and functions. This standard was published as [KRM Functions Specification][spec]
and donated to the CNCF as part of the Kubernetes SIG-CLI.

Understanding this specification enables you to have a deeper understanding of
how things work under the hood. It also enables to create your own toolchain for
function development if you so desire.

As an example, you can see the `ResourceList` containing resources in the
`wordpress` package:

```shell
$ kpt fn source wordpress | less
```

[spec]:
  https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md#krm-functions-specification
