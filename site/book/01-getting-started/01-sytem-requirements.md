In order to follow along with the examples in this book, the following needs to be installed on you
system:

## kpt

[Install the kpt CLI][install] and ensure you are running **version 1.0 or later**:

```shell
$ kpt version
```

TODO: Clean up installation page.
TODO: kpt 1.0 binaries needs to be available.

## Docker

kpt requires that you have [Docker] installed and configured.

## Kubernetes cluster

In order to deploy the examples, you need a Kubernetes cluster and a configured kubeconfig context.

For testing purposes, [kind] tool is useful for running ephemeral Kubernetes cluster on your local host.

[install]: /installation/
[docker]: https://docs.docker.com/get-docker/
[kind]: https://kind.sigs.k8s.io/docs/user/quick-start/
