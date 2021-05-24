---
title: "`live`"
linkTitle: "live"
weight: 3
type: docs
description: >
   Deploy local packages to a cluster.
---
<!--mdtogo:Short
    Deploy local packages to a cluster.
-->

<!--mdtogo:Long-->
The `live` command group contains subcommands for deploying local
`kpt` packages to a cluster.
<!--mdtogo-->


#### Flags

These are the same flags as is available in [kubectl].

```
--as:
  Username to impersonate for the operation.

--as-group:
  Group to impersonate for the operation, this flag can be repeated to specify multiple groups.

--cache-dir:
  Default cache directory (default "/Users/mortent/.kube/cache").

--certificate-authority:
  Path to a cert file for the certificate authority.

--client-certificate:
  Path to a client certificate file for TLS.

--client-key:
  Path to a client key file for TLS.

--cluster:
  The name of the kubeconfig cluster to use.

--context:
  The name of the kubeconfig context to use.

--insecure-skip-tls-verify:
  If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure.

--kubeconfig:
  Path to the kubeconfig file to use for CLI requests.

--namespace:
  If present, the namespace scope for this CLI request.

--password:
  Password for basic authentication to the API server.

--request-timeout:
  The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0").

--server:
  The address and port of the Kubernetes API server.

--tls-server-name:
  Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used.

--token:
  Bearer token for authentication to the API server.

--user:
  The name of the kubeconfig user to use.

--username:
   Username for basic authentication to the API server.
```

[kubectl]: https://kubernetes.io/docs/reference/kubectl/kubectl/#options
