#!/usr/bin/env bash
# Copyright 2022 The kpt Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Stricter error handling
set -e # Exit on error
set -u # Must predefine variables
set -o pipefail # Check errors in piped commands

PORCH_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

function error() {
  cat <<EOF
Error: ${1}
Usage: ${0} [flags]
Supported Flags:
  --destination DIRECTORY             ... directory in which to create the Porch deploymetn blueprint
  --project GCP_PROJECT               ... ID of GCP project in which Porch will be deployed; if set, will
                                          customize deploymetn service accounts
  --server-image IMAGE                ... address of the Porch server image
  --controllers-image IMAGE           ... address of the Porch controllers image
  --function-image IMAGE              ... address of the Porch function runtime image
  --wrapper-server-image IMAGE        ... address of the Porch function wrapper server image
  --server-sa SVC_ACCOUNT             ... GCP service account to run the Porch server
  --controllers-sa SVC_ACCOUNT        ... GCP service account to run the Porch Controllers workload
  --enabled-reconcilers RECONCILDERS  ... comma-separated list of reconcilers that should be enabled in
                                          porch controller
EOF
  exit 1
}

# Flag variables
DESTINATION=""
SERVER_IMAGE=""
CONTROLLERS_IMAGE=""
FUNCTION_IMAGE=""
WRAPPER_SERVER_IMAGE=""
SERVER_SA=""
CONTROLLERS_SA=""
FUNCTION_RUNNER_SA=""
PROJECT=""
ENABLED_RECONCILERS=""

while [[ $# -gt 0 ]]; do
  key="${1}"
  case "${key}" in
    --destination)
      DESTINATION="${2}"
      shift 2
    ;;

    --project)
      PROJECT="${2}"
      shift 2
    ;;

    --server-image)
      SERVER_IMAGE="${2}"
      shift 2
    ;;

    --controllers-image)
      CONTROLLERS_IMAGE="${2}"
      shift 2
    ;;

    --function-image)
      FUNCTION_IMAGE="${2}"
      shift 2
    ;;

    --wrapper-server-image)
      WRAPPER_SERVER_IMAGE="${2}"
      shift 2
    ;;

     --server-sa)
      SERVER_SA="${2}"
      shift 2
      ;;

    --controllers-sa)
      CONTROLLERS_SA="${2}"
      shift 2
      ;;

    --enabled-reconcilers)
      ENABLED_RECONCILERS="${2}"
      shift 2
      ;;

    *)
      error "Invalid argument: ${key}"
    ;;
  esac
done

# Defaults

if [ -n "${PROJECT}" ]; then
  FUNCTION_RUNNER_SA="${FUNCTION_RUNNER_SA:-iam.gke.io/gcp-service-account=porch-function-runner@${PROJECT}.iam.gserviceaccount.com}"
  CONTROLLERS_SA="${CONTROLLERS_SA:-iam.gke.io/gcp-service-account=porch-sync@${PROJECT}.iam.gserviceaccount.com}"
  SERVER_SA="${SERVER_SA:-iam.gke.io/gcp-service-account=porch-server@${PROJECT}.iam.gserviceaccount.com}"
fi

echo ${CONTROLLERS_SA}
echo ${SERVER_SA}

function validate() {
  [ -n "${DESTINATION}"       ] || error "--destination is required"
  [ -n "${SERVER_IMAGE}"      ] || error "--server-image is required"
  [ -n "${CONTROLLERS_IMAGE}" ] || error "--controllers-image is required"
  [ -n "${FUNCTION_IMAGE}"    ] || error "--function-image is required"
}

function customize-image {
  local OLD="${1}"
  local NEW="${2}"
  local TAG="${NEW##*:}"
  local IMG="${NEW%:*}"

  kpt fn eval "${DESTINATION}" --image set-image:v0.1.1 -- \
    "name=${OLD}" \
    "newName=${IMG}" \
    "newTag=${TAG}"
}

function customize-image-in-env {
  local OLD="${1}"
  local NEW="${2}"
  local TAG="${NEW##*:}"
  local IMG="${NEW%:*}"
  local FN_CONFIG="${PORCH_DIR}/.build/set-image-config.yaml"

  cat > "${FN_CONFIG}" << EOF
apiVersion: fn.kpt.dev/v1alpha1
kind: SetImage
metadata:
  name: my-func-config
image:
  name: ${OLD}
  newName: ${IMG}
  newTag: ${TAG}
additionalImageFields:
- group: apps
  version: v1
  kind: Deployment
  path: spec/template/spec/containers[]/env[]/value
EOF

  trap "rm -f ${FN_CONFIG}" EXIT

  kpt fn eval "${DESTINATION}" --image set-image:v0.1.1 --fn-config "${FN_CONFIG}" || echo "kpt fn eval failed"
}

function customize-sa {
  local NAME="${1}"
  local SA="${2}"

  kpt fn eval "${DESTINATION}" --image set-annotations:v0.1.4 \
    --match-api-version=v1 \
    --match-kind=ServiceAccount \
    "--match-name=${NAME}" \
    --match-namespace=porch-system -- \
    "${SA}"
}

function customize-container-env {
  local ENV_KEY="${1}"
  local ENV_VAL="${2}"

  # TODO: This is terrible. Do we have a good way to handle this with kpt?
  sed "/env:/a\        - name: ${ENV_KEY}\n          value: ${ENV_VAL}\n" -i "${DESTINATION}/9-controllers.yaml"
}

function main() {
  # Repository CRD
  cp "./api/porchconfig/v1alpha1/config.porch.kpt.dev_functions.yaml" \
     "${DESTINATION}/0-functions.yaml"
  cp "./api/porchconfig/v1alpha1/config.porch.kpt.dev_repositories.yaml" \
     "${DESTINATION}/0-repositories.yaml"
  cp "./internal/api/porchinternal/v1alpha1/config.porch.kpt.dev_packagerevs.yaml" \
     "${DESTINATION}/0-packagerevs.yaml"

  # Porch Deployment Config
  cp ${PORCH_DIR}/deployments/porch/*.yaml "${PORCH_DIR}/deployments/porch/Kptfile" "${DESTINATION}"
  # Copy Porch controller manager rbac
  cp ${PORCH_DIR}/controllers/config/rbac/role.yaml "${DESTINATION}/9-porch-controller-clusterrole.yaml"

  IFS=',' read -ra RECONCILERS <<< "$ENABLED_RECONCILERS"
  for i in "${RECONCILERS[@]}"; do
    if [[ -f "${PORCH_DIR}/controllers/config/crd/bases/config.porch.kpt.dev_${i}.yaml" ]]; then
      # Copy over the CRD (if it exists)
      cp "${PORCH_DIR}/controllers/config/crd/bases/config.porch.kpt.dev_${i}.yaml" \
         "${DESTINATION}/0-${i}.yaml"
    fi
    # Update the porch-controllers Deployment env variables to enable the reconciler.
    customize-container-env \
      "ENABLE_${i^^}" \
      "\"true\""
    # Copy over the rbac rules for the reconciler
    cp "${PORCH_DIR}/controllers/${i}/config/rbac/role.yaml" \
    "${DESTINATION}/9-porch-controller-${i}-clusterrole.yaml"
    # Copy over the rbac rules for the reconciler
    cp "${PORCH_DIR}/controllers/${i}/config/rbac/rolebinding.yaml" \
    "${DESTINATION}/9-porch-controller-${i}-clusterrolebinding.yaml"
  done

  customize-image \
    "gcr.io/example-google-project-id/porch-function-runner:latest" \
    "${FUNCTION_IMAGE}"
  customize-image \
    "gcr.io/example-google-project-id/porch-server:latest" \
    "${SERVER_IMAGE}"
  customize-image \
    "gcr.io/example-google-project-id/porch-controllers:latest" \
    "${CONTROLLERS_IMAGE}"
  customize-image-in-env \
    "gcr.io/example-google-project-id/porch-wrapper-server:latest" \
    "${WRAPPER_SERVER_IMAGE}"

  if [ -n "${CONTROLLERS_SA}" ]; then
    customize-sa "porch-controllers" "${CONTROLLERS_SA}"
  fi

  if [ -n "${SERVER_SA}" ]; then
    customize-sa "porch-server" "${SERVER_SA}"
  fi

  if [ -n "${FUNCTION_RUNNER_SA}" ]; then
    # TODO: Rename serviceaccount for consistency (avoid abbreviations?)
    customize-sa "porch-fn-runner" "${FUNCTION_RUNNER_SA}"
  fi
}

validate
main
