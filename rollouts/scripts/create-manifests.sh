#! /usr/bin/env bash

# Stricter error handling
set -e # Exit on error
set -u # Must predefine variables
set -o pipefail # Check errors in piped commands

ROLLOUTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

function error() {
  cat <<EOF
Error: ${1}
Usage: ${0} [flags]
Supported Flags:
  --destination DIRECTORY             ... directory in which to create the Rollout deploymetn kpt pkg
  --controller-image IMAGE           ... address of the Porch controllers image
EOF
  exit 1
}

# Flag variables
DESTINATION=""
CONTROLLER_IMAGE=""

while [[ $# -gt 0 ]]; do
  key="${1}"
  case "${key}" in
    --destination)
      DESTINATION="${2}"
      shift 2
    ;;

    --controller-image)
      CONTROLLER_IMAGE="${2}"
      shift 2
    ;;

    *)
      error "Invalid argument: ${key}"
    ;;
  esac
done

# Defaults
DESTINATION="manifests"

function validate() {
  [ -n "${DESTINATION}"       ] || error "--destination is required"
  [ -n "${CONTROLLER_IMAGE}" ] || error "--controller-image is required"
}

function log() {
    echo $1
}

echo ${DESTINATION}
echo ${CONTROLLER_IMAGE}

# function to generate CRDs for rollouts APIs
function generate_crds {
    log "generating crds..."
    kustomize build config/crd > ${DESTINATION}/crds/crds.yaml
}

# function to generate manifests for deploying rollouts controller
function generate_controller_manifests {
    log "generating controller manifests..."
    kustomize build config/default > ${DESTINATION}/controller/controller.yaml
}

# update the controller image
function set_controller_image {
  local OLD="controller:latest"
  local NEW="${1}"
  local TAG="${NEW##*:}"
  local IMG="${NEW%:*}"
  kpt fn eval ${DESTINATION}/controller --image set-image:v0.1.1 -- \
    "name=${OLD}" \
    "newName=${IMG}" \
    "newTag=${TAG}"
}

validate && \
generate_crds && \
generate_controller_manifests && \
set_controller_image ${CONTROLLER_IMAGE}