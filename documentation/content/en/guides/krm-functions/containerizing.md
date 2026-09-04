---
title: Containerizing
linkTitle: Containerizing
description: Package a KRM function as a container image.
toc_hide: false
menu:
  main:
    parent: "KRM Function Developer Guide"
    weight: 40
---
KRM functions are distributed as container images. This guide covers building
and running containerized functions.

## Dockerfile

The [krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog)
provides a shared Dockerfile at `build/docker/go/Dockerfile` that all the catalog
functions use. It accepts `BUILDER_IMAGE` and `BASE_IMAGE` as build args.

For standalone functions or local development, use a multi-stage build with a
minimal base image. The function binary should be statically linked (no CGO), so
it can run on `scratch` or `distroless`:

```dockerfile
FROM golang:1.26-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/function ./

FROM scratch
COPY --from=builder /usr/local/bin/function /usr/local/bin/function
ENTRYPOINT ["function"]
```

Key points:
- `CGO_ENABLED=0` produces a static binary that runs on `scratch`.
- The `scratch` base image has zero overhead — no shell, no OS packages.
- If you need TLS certificates (e.g., for network calls), use `gcr.io/distroless/static` instead of `scratch`.
- Copy only the binary to the final image to minimize size.

### Alternative with distroless

```dockerfile
FROM golang:1.26-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/function ./

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /usr/local/bin/function /usr/local/bin/function
ENTRYPOINT ["function"]
```

## Building

```bash
docker build -t ghcr.io/kptdev/krm-functions-catalog/my-function:v0.1 .
```

### Image Naming Convention

Follow this pattern for function images:

```
ghcr.io/kptdev/krm-functions-catalog/{function-name}:{version}
```

Examples:
- `ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.1`
- `ghcr.io/kptdev/krm-functions-catalog/enforce-namespace:v1.0`
- `ghcr.io/kptdev/krm-functions-catalog/generate-configmap:v0.3`

Use semantic versioning for tags. Avoid `latest` in production pipelines.

## Running

KRM functions read from STDIN and write to STDOUT:

```bash
docker run --rm -i ghcr.io/kptdev/krm-functions-catalog/my-function:v0.1 < input.yaml > output.yaml
```

### With file mode

```bash
docker run --rm -v $(pwd):/data ghcr.io/kptdev/krm-functions-catalog/my-function:v0.1 /data/deployment.yaml
```

Note: file mode assembles the given files into a ResourceList with an **empty
functionConfig**. Functions that require configuration should be run via STDIN
(or a `kpt` pipeline) so the functionConfig is provided.

### Help and doc flags

```bash
docker run --rm ghcr.io/kptdev/krm-functions-catalog/my-function:v0.1 --help
docker run --rm ghcr.io/kptdev/krm-functions-catalog/my-function:v0.1 --doc
```

## Using with kpt

In a `Kptfile` pipeline, `kpt fn render` will pull the image from the registry
and run it against your package resources:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.1
      configMap:
        app: my-app
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/enforce-namespace:v1.0
      configMap:
        namespace: production
```

Note: the image must be published and accessible from the machine running
`kpt fn render`. For local development, build the image locally first. It
will be used from the local Docker cache without pulling.

## Tips

- Keep images small — a typical Go KRM function image is 5–15 MB with `scratch`.
- Pin dependency versions in `go.mod` for reproducible builds.
- Use `.dockerignore` to exclude test data, docs, and other non-build files.
- Test the container locally before publishing:
  ```bash
  echo '{"apiVersion":"config.kubernetes.io/v1","kind":"ResourceList","items":[]}' | \
    docker run --rm -i ghcr.io/kptdev/krm-functions-catalog/my-function:v0.1
  ```

## Publishing

Publishing function images to a registry is handled by the
[krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog)
CI pipeline. See the catalog's
[CONTRIBUTING.md](https://github.com/kptdev/krm-functions-catalog/blob/main/CONTRIBUTING.md)
for the release workflow.