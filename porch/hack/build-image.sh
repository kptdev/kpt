#!/usr/bin/env bash
# Copyright 2022 Google LLC
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



BASE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." && pwd )"

# Parse arguments

TAG="${TAG:-latest}"
PROJECT="${PROJECT}"
REPOSITORY="${REPOSITORY}"

while [[ $# -gt 0 ]]; do
  key="${1}"

  case "${key}" in
    --project)
      PROJECT="${2}"
      shift 2
    ;;

    --project=*)
      PROJECT="${key#*=}"
      shift
    ;;

    --tag)
      TAG="${2}"
      shift 2
    ;;

    --tag=*)
      TAG="${key#*=}"
      shift
    ;;

    --repository)
      REPOSITORY="${2}"
      shift 2
    ;;

    --repository=*)
      REPOSITORY="${key#*=}"
      shift
    ;;

    --push)
      PUSH=Yes
      shift
    ;;

    *)
      echo "Invalid argument: ${key}"
      exit 1
    ;;

  esac
done

function error() {
  echo $1
  cat <<EOF
Usage: build-image.sh [flags]
Supported Flags:
  --project [GCP_PROJECT]   ... will build image gcr.io/{GCP_PROJECT}/porch:${TAG}
  --tag [TAG]               ... tag for the image, i.e. 'latest'
  --repository [REPOSITORY] ... the image repository. will build image
                                [REPOSITORY]/porch:${TAG}
  --push                    ... push the image to the repository also
EOF
  exit 1
}

function run() {
  echo "$@"
  $@
}

if [[ -z "${REPOSITORY}" ]]; then
  [[ -n "${PROJECT}" ]] || error "--project or --repository is required"
  REPOSITORY="gcr.io/${PROJECT}"  
fi

IMAGE="${REPOSITORY}/porch:${TAG}"

[[ "${PUSH}" == "Yes" ]] || run docker buildx build --load -t "${IMAGE}" -f "${BASE_DIR}/porch/hack/Dockerfile" "${BASE_DIR}"
[[ "${PUSH}" != "Yes" ]] || run docker buildx build --push -t "${IMAGE}" -f "${BASE_DIR}/porch/hack/Dockerfile" "${BASE_DIR}"
