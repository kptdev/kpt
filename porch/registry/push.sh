#!/bin/bash

set -e
set -x

GCP_PROJECT_ID=$(shell gcloud config get-value project)

pushd crds/fn.kpt.dev/renderhelmchart/v1alpha1
  rm -f layer.tar.gz
  tar cvf layer.tar.gz *.yaml
  IMG="us-west1-docker.pkg.dev/${GCP_PROJECT_ID}/registry/crds/fn.kpt.dev/renderhelmchart:v1alpha1"
  crane append -f layer.tar.gz -t "${IMG}"
  crane mutate --annotation kpt.dev/function=gcr.io/kpt-fn/render-helm-chart:unstable "${IMG}"
  rm -f layer.tar.gz
 popd
