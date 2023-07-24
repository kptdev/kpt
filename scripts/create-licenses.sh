#!/usr/bin/env bash
# Copyright 2015 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Update the LICENSES document.
# Generates a table of Godep dependencies and their license.
#
# Usage:
#    $0 [--create-missing] [/path/to/licenses]
#
#    --create-missing will write the files that only exist upstream, locally.
#    This option is mostly used for testing as we cannot check-in any of the
#    additionally created files into the vendor auto-generated tree.
#
#    Run every time a license file is added/modified within /vendor to
#    update LICENSES

set -o errexit
set -o nounset
set -o pipefail

export LANG=C
export LC_ALL=C

# Name of file to write licenses.
VENDOR_LICENSE_FILE="LICENSES.txt"

# Zip package name for Mozilla licensed source code.
ZIP_FILENAME="lib.zip"

###############################################################################
# Process package content
#
# @param package  The incoming package name
# @param type     The type of content (LICENSE, COPYRIGHT or COPYING)
#
process_content () {
  local package=$1
  local type=$2

  local package_root
  local ensure_pattern
  local dir_root
  local find_maxdepth
  local find_names
  local -a local_files=()

  # Necessary to expand {}
  case ${type} in
      LICENSE) find_names=(-iname 'licen[sc]e*')
               find_maxdepth=1
               # Sadly inconsistent in the wild, but mostly license files
               # containing copyrights, but no readme/notice files containing
               # licenses (except to "see license file")
               ensure_pattern="license|copyright"
               ;;
    # We search READMEs for copyrights and this includes notice files as well
    # Look in as many places as we find files matching
    COPYRIGHT) find_names=(-iname 'notice*' -o -iname 'readme*')
               find_maxdepth=3
               ensure_pattern="copyright"
               ;;
      COPYING) find_names=(-iname 'copying*')
               find_maxdepth=1
               ensure_pattern="license|copyright"
               ;;
  esac

  # Start search at package root
  case ${package} in
    github.com/*|golang.org/*|bitbucket.org/*|gonum.org/*)
     package_root=$(echo "${package}" |awk -F/ '{ print $1"/"$2"/"$3 }')
     ;;
    go4.org/*)
     package_root=$(echo "${package}" |awk -F/ '{ print $1 }')
     ;;
    gopkg.in/*)
     # Root of gopkg.in package always ends with '.v(number)' and my contain
     # more than two path elements. For example:
     # - gopkg.in/yaml.v2
     # - gopkg.in/inf.v0
     # - gopkg.in/square/go-jose.v2
     package_root=$(echo "${package}" |grep -oh '.*\.v[0-9]')
     ;;
    */*)
     package_root=$(echo "${package}" |awk -F/ '{ print $1"/"$2 }')
     ;;
    *)
     package_root="${package}"
     ;;
  esac

  # Find files - only root and package level
  local_files=()
  IFS=" " read -r -a local_files <<< "$(
    for dir_root in ${package} ${package_root}; do
      [[ -d ${DEPS_DIR}/${dir_root} ]] || continue
      
      # One (set) of these is fine
      find "${DEPS_DIR}/${dir_root}" \
          -xdev -follow -maxdepth ${find_maxdepth} \
          -type f "${find_names[@]}"
    done | sort -u)"

  local index
  local f
  index="${package}-${type}"
  if [[ -z "${CONTENT[${index}]-}" ]]; then
    for f in "${local_files[@]-}"; do
      if [[ -z "$f" ]]; then
        # Set the default value and then check it to prevent
        # accessing potentially empty array
        continue
      fi
      # Find some copyright info in any file and break
      if grep -E -i -wq "${ensure_pattern}" "${f}"; then
        CONTENT[${index}]="${f}"
        break
      fi
    done
  fi
}


#############################################################################
# MAIN
#############################################################################
KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${KUBE_ROOT}/scripts/lib/init.sh"

export GO111MODULE=on

# Check bash version
if (( BASH_VERSINFO[0] < 4 )); then
  echo
  echo "ERROR: Bash v4+ required."
  # Extra help for OSX
  if [[ "$(uname -s)" == "Darwin" ]]; then
    echo
    echo "Ensure you are up to date on the following packages:"
    echo "$ brew install md5sha1sum bash jq"
  fi
  echo
  exit 9
fi

TMP_LICENSE_FILE="/tmp/LICENSES.$$"
DEPS_DIR="vendor"
declare -Ag CONTENT

# Run go mod vendor to create vendor dependencies.
go mod vendor

# Some cleanups
# Rename all license files to `LICENSE`
find vendor -type f -name LICENSE.md -execdir mv LICENSE.md LICENSE ';'
find vendor -type f -name LICENSE.txt -execdir mv LICENSE.txt LICENSE ';'
# Move LICENSE file to root directory of each dependency. This is necessary for
# dependencies where code is stored in a versioned sub-directory.
V2_LICENSE_DIR="vendor/github.com/cpuguy83/go-md2man"
mv ${V2_LICENSE_DIR}/v2/LICENSE ${V2_LICENSE_DIR}
GO_RESTFUL_LICENSE_DIR="vendor/github.com/emicklei/go-restful"
mv ${GO_RESTFUL_LICENSE_DIR}/v3/LICENSE ${GO_RESTFUL_LICENSE_DIR}
KLOG_LICENSE_DIR="vendor/k8s.io/klog"
mv ${KLOG_LICENSE_DIR}/v2/LICENSE ${KLOG_LICENSE_DIR}
XXHASH_LICENSE_DIR="vendor/github.com/cespare/xxhash"
mv "${XXHASH_LICENSE_DIR}/v2/LICENSE" "${XXHASH_LICENSE_DIR}"
BLACKFRIDAY_LICENSE_DIR="vendor/github.com/russross/blackfriday"
mv "${BLACKFRIDAY_LICENSE_DIR}/v2/LICENSE" "${BLACKFRIDAY_LICENSE_DIR}"

# Loop through every vendored package
mozilla_repos=()
for PACKAGE in $(go list -mod=mod -m -json all | jq -r .Path | sort -f); do
  if [[ -e "staging/src/${PACKAGE}" ]]; then
    # echo "$PACKAGE is a staging package, skipping" > /dev/stderr
    continue
  fi
  if [[ ! -e "${DEPS_DIR}/${PACKAGE}" ]]; then
    # echo "$PACKAGE doesn't exist in vendor, skipping" > /dev/stderr
    continue
  fi
  # TODO: samwronski - remove this edge case
  # The above if statement skips dependencies which did not get checked out with
  # `go mod vendor` however that does not catch this edge case.
  # Kpt currently depends on 2 versions of posener. Because v2 *is* checked out
  # the directory does exist causing the above check to pass. However this repo
  # is not included in the vendor directory so a license will not be found.
  if [[ "${PACKAGE}" == "github.com/posener/complete" ]]; then
    continue
  fi

  # Skip self.
  # The LICENSE file is at the root but vendor directory contents are selective.
  if [[ "${PACKAGE}" =~ ^github.com/GoogleContainerTools/kpt(/.*)?$ ]]; then
    # echo "Skipping ${PACKAGE}" > /dev/stderr
    continue
  fi

  # cloud.google.com/go has a tricky structure in terms of LICENSE files.
  # Use the go.mod package path to resolve the license.
  if [[ "${PACKAGE}" =~ ^cloud.google.com/go(/.*)?$ ]]; then
    # Make sure we have downloaded the package into the package cache
    go mod download "${PACKAGE}"
    # Look for a license in the package cache
    for PACKAGE_DIR in $(go list -mod=mod -m -json ${PACKAGE} | jq -r .Dir); do
      if [[ -e "${PACKAGE_DIR}/LICENSE" ]]; then
        CONTENT["${PACKAGE}-LICENSE"]="${PACKAGE_DIR}/LICENSE"
      fi
    done
  fi

  process_content "${PACKAGE}" LICENSE
  process_content "${PACKAGE}" COPYRIGHT
  process_content "${PACKAGE}" COPYING

  # display content
  echo
  echo "================================================================================"
  echo "= ${DEPS_DIR}/${PACKAGE} licensed under: ="
  echo

  file=""
  if [[ -n "${CONTENT[${PACKAGE}-LICENSE]-}" ]]; then
      file="${CONTENT[${PACKAGE}-LICENSE]-}"
  elif [[ -n "${CONTENT[${PACKAGE}-COPYRIGHT]-}" ]]; then
      file="${CONTENT[${PACKAGE}-COPYRIGHT]-}"
  elif [[ -n "${CONTENT[${PACKAGE}-COPYING]-}" ]]; then
      file="${CONTENT[${PACKAGE}-COPYING]-}"
  fi
  if [[ -z "${file}" ]]; then
      cat > /dev/stderr << __EOF__
No license could be found for ${PACKAGE} - aborting.

Options:
1. Check if the upstream repository has a newer version with LICENSE, COPYRIGHT and/or
   COPYING files.
2. Contact the author of the package to ensure there is a LICENSE, COPYRIGHT and/or
   COPYING file present.
3. Do not use this package in Kubernetes.
__EOF__
      exit 9
  fi

  # Check to see if its a Mozilla license. If so, we need to package the source code.
  license=$(cat "${file}")
  if [[ "$license" == *"Mozilla"* ]]
  then
    mozilla_repos+=("${DEPS_DIR}/${PACKAGE}")
  fi

  cat ${file}

  echo
  echo "= ${file} $(kube::util::md5 "${file}")"
  echo "================================================================================"
  echo
done >> ${TMP_LICENSE_FILE}

cat ${TMP_LICENSE_FILE} > ${VENDOR_LICENSE_FILE}

# initialize zip file to ensure existence (downstream builds depend on libs.zip)
README="${KUBE_ROOT}/scripts/docs/create-licenses/README.md"
zip -q "${ZIP_FILENAME}" "${README}"

# Create a package of Mozilla repository source code (only go code).
[ ${#mozilla_repos[@]} != 0 ] && zip -qur "${ZIP_FILENAME}" "${mozilla_repos[@]}" -i '*.go'

# Cleanup vendor directory
rm -rf vendor
