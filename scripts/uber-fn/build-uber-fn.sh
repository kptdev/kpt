#!/usr/bin/env bash

set_labels_image=gcr.io/kpt-fn/set-labels:v0.1.4
set_namespace_image=gcr.io/kpt-fn/set-namespace:v0.1.3

# extract set-labels binary
container_id=`docker create $set_labels_image`
docker cp $container_id:/usr/local/bin/function ./set-labels
docker rm $container_id

# extract set-namespace binary
container_id=`docker create $set_namespace_image`
docker cp $container_id:/usr/local/bin/function ./set-namespace
docker rm $container_id

# package extracted binaries in a new container image
docker build -t gcr.io/kpt-fn-demo/all_fns:v0.0.1 .

# clean if up
rm -f ./set-namespace ./set-labels
