#! /bin/bash

# Intentionally tag a wrong image as gcr.io/kpt-fn/set-annotations:v0.1
docker pull gcr.io/kpt-fn/set-labels:v0.1
docker tag gcr.io/kpt-fn/set-labels:v0.1 gcr.io/kpt-fn/set-annotations:v0.1
