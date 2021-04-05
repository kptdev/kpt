---
title: "docker"
linkTitle: "docker"
weight: 4
type: docs
description: >
    Run kpt in a docker container.
---

Use one of the kpt docker images.

| Feature   |:`kpt`:|:`kpt-gcloud`:|
| --------- |:-----:|:------------:|
| kpt       | X     | X            |
| git       | X     | X            |
| diffutils | X     | X            |
| gcloud    |       | X            |

## [gcr.io/kpt-dev/kpt]

```shell
docker run gcr.io/kpt-dev/kpt version
```

## [gcr.io/kpt-dev/kpt-gcloud]

An image which includes kpt based upon the Google [cloud-sdk] alpine image.

```shell
docker run gcr.io/kpt-dev/kpt-gcloud version
```

[gcr.io/kpt-dev/kpt]: https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt?gcrImageListsize=30

[gcr.io/kpt-dev/kpt-gcloud]: https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt-gcloud?gcrImageListsize=30

[cloud-sdk]: https://github.com/GoogleCloudPlatform/cloud-sdk-docker