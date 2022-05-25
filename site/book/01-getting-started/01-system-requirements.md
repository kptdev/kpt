In order to follow along with the examples in this book, the following needs to be installed on your
system:

## kpt

[Install the kpt CLI][install] and ensure you are running **version 1.0 or later**:

```shell
$ kpt version
```

## Git

kpt requires that you have [Git] installed and configured.

## Container Runtime

`kpt` requires you to have at least one of the following runtimes installed and configured.

Environment variable can be used to control which container runtime to use. More
details can be found in the reference documents for [`kpt fn render`](/reference/cli/fn/render/)
and [`kpt fn eval`](/reference/cli/fn/eval/).

### Docker

Here is the [instructions][install-docker] to have Docker installed and configured.

### Podman

Here is the [instructions][install-podman] to have Podman installed and configured.

If you want to set up rootless container runtime, [this][rootless] may be a
useful resource for you.

## Kubernetes cluster

In order to deploy the examples, you need a Kubernetes cluster and a configured kubeconfig context.

For testing purposes, [kind] tool is useful for running ephemeral Kubernetes cluster on your local host.

[install]: /installation/
[install-docker]: https://docs.docker.com/get-docker/
[install-podman]: https://podman.io/getting-started/installation
[rootless]: https://rootlesscontaine.rs/
[git]: https://git-scm.com/book/en/v2/Getting-Started-Installing-Git
[kind]: https://kind.sigs.k8s.io/docs/user/quick-start/
