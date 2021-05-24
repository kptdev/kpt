#! /bin/bash

# Intentionally tag a wrong image as gcr.io/kpt-fn/set-labels:v0.1, since we
# expect the correct image to be pulled and override the wrong image.
docker pull gcr.io/kpt-fn/starlark:v0.1
docker tag gcr.io/kpt-fn/starlark:v0.1 gcr.io/kpt-fn/set-labels:v0.1
