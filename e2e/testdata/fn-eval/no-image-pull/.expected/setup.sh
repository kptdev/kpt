#!/bin/bash

docker image inspect gcr.io/kpt-fn/search-replace:v0.1
# if inspect exits with a 0 exit code the image was found locally, remove it
if [[ $? == 0 ]]; then
    docker image rm gcr.io/kpt-fn/search-replace:v0.1
fi
