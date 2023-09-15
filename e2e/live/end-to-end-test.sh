#!/usr/bin/env bash
###########################################################################
# Copyright 2020 The kpt Authors
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
#
# Description:
#   Validates multiple scenarios for kpt live commands.
#
# How to use this script:
#   FROM KPT ROOT DIR: ./e2e/live/end-to-end-test.sh
#
# Example KPT ROOT DIR:
#   ~/go/src/github.com/GoogleContainerTools/kpt
#
# Flags:
#   -b) Build the kpt binary with dependencies at HEAD. Downloads HEAD
#       for cli-utils and kustomize repositories, and builds kpt using
#       these downloaded repositories.
#   -k) Run against this Kubernetes version. Creates a local Kubernetes
#       cluster at this version using "kind". Only accepts MAJOR.MINOR.
#       Example: 1.17. This is translated to a patch version. Example:
#       1.17 runs version 1.17.11.
#
# Prerequisites (must be in $PATH):
#   kind - Kubernetes in Docker
#   kubectl - version of kubectl should be within +/- 1 version of cluster.
#     CHECK: $ kubectl version
#
# Examples:
#   $ ./e2e/live/end-to-end-test.sh -b
#   $ ./e2e/live/end-to-end-test.sh -k 1.17
#   $ ./e2e/live/end-to-end-test.sh -bk 1.17
#
###########################################################################

# TODO
#  1) Validate prerequisites (e.g. kind, kubectl)
#  2) Refactor helper functions into another file
#  3) Fix -k <UNKNOWN K8S VERSION>
#  4) Add "-v" verbosity level flags
#  5) Count the test cases and print it out
#  6) Print timing for the tests


###########################################################################
#  Parameters/args
###########################################################################

# A POSIX variable; reset in case getopts has been used previously in the shell.
OPTIND=1

# Kind/Kubernetes versions.
KIND_1_28_VERSION=1.28.0@sha256:b7a4cad12c197af3ba43202d3efe03246b3f0793f162afb40a33c923952d5b31
KIND_1_27_VERSION=1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72
KIND_1_26_VERSION=1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb
KIND_1_25_VERSION=1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
KIND_1_24_VERSION=1.24.15@sha256:7db4f8bea3e14b82d12e044e25e34bd53754b7f2b0e9d56df21774e6f66a70ab
KIND_1_23_VERSION=1.23.17@sha256:59c989ff8a517a93127d4a536e7014d28e235fb3529d9fba91b3951d461edfdb
KIND_1_22_VERSION=1.22.17@sha256:f5b2e5698c6c9d6d0adc419c0deae21a425c07d81bbf3b6a6834042f25d4fba2
KIND_1_21_VERSION=1.21.14@sha256:8a4e9bb3f415d2bb81629ce33ef9c76ba514c14d707f9797a01e3216376ba093

DEFAULT_K8S_VERSION=${KIND_1_28_VERSION}

# Change from empty string to build the kpt binary from the downloaded
# repositories at HEAD, including dependencies cli-utils and kustomize.
BUILD_DEPS_AT_HEAD=""

# Default Kubernetes cluster version to run test against.
K8S_VERSION=${DEFAULT_K8S_VERSION}

HAS_TEST_FAILURE=0

# Parse/validate parameters
options="bk:"
while getopts $options opt; do
    case "$opt" in
	b)  BUILD_DEPS_AT_HEAD=1;;
	k)  short_version=$OPTARG
	    case "$short_version" in
		1.21) K8S_VERSION=$KIND_1_21_VERSION
		      ;;
		1.22) K8S_VERSION=$KIND_1_22_VERSION
		      ;;
		1.23) K8S_VERSION=$KIND_1_23_VERSION
		      ;;
		1.24) K8S_VERSION=$KIND_1_24_VERSION
		      ;;
		1.25) K8S_VERSION=$KIND_1_25_VERSION
		      ;;
		1.26) K8S_VERSION=$KIND_1_26_VERSION
		      ;;
		1.27) K8S_VERSION=$KIND_1_27_VERSION
		      ;;
		1.28) K8S_VERSION=$KIND_1_28_VERSION
		      ;;
        *) K8S_VERSION=$short_version
		      ;;
	    esac
	    ;;
	\? ) echo "Usage: $0 [-b] [-k k8s-version]" >&2; exit 1;;
    esac
done

shift $((OPTIND-1))

[ "${1:-}" = "--" ] && shift

###########################################################################
#  Colors
###########################################################################

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

###########################################################################
#  Helper functions
###########################################################################

function downloadPreviousKpt {
  set -e
  KPT_VERSION=0.39.2
  echo "Downloading v${KPT_VERSION} kpt binary..."
  uname="$(uname -s)"
  if [[ "$uname" == "Linux" ]]
  then
    echo "Running on Linux"
    curl -LJ -o kpt.tar.gz https://github.com/GoogleContainerTools/kpt/releases/download/v${KPT_VERSION}/kpt_linux_amd64-${KPT_VERSION}.tar.gz > $OUTPUT_DIR/kptdownload 2>&1
  elif [[ "$uname" == "Darwin" ]]
  then
    echo "Running on Darwin"
    curl -LJ -o kpt.tar.gz https://github.com/GoogleContainerTools/kpt/releases/download/v${KPT_VERSION}/kpt_darwin_amd64-${KPT_VERSION}.tar.gz > $OUTPUT_DIR/kptdownload 2>&1
  else
    echo -e "${RED}ERROR${NC}: Unknown OS $uname"
    exit 1
  fi
  tar -xvf kpt.tar.gz > $OUTPUT_DIR/kptdownload 2>&1
  mv kpt $BIN_DIR/previouskpt
  echo -e "Downloading previous kpt binary...${GREEN}SUCCESS${NC}"
  rm kpt.tar.gz LICENSES.txt lib.zip
  set +e
}

function downloadKpt1.0 {
  set -e
  KPT_VERSION=1.0.0-beta.13
  echo "Downloading v${KPT_VERSION} kpt binary..."
  uname="$(uname -s)"
  if [[ "$uname" == "Linux" ]]
  then
    echo "Running on Linux"
    curl -LJ -o kpt.tar.gz https://github.com/GoogleContainerTools/kpt/releases/download/v${KPT_VERSION}/kpt_linux_amd64-${KPT_VERSION}.tar.gz > $OUTPUT_DIR/kptdownload 2>&1
  elif [[ "$uname" == "Darwin" ]]
  then
    echo "Running on Darwin"
      curl -LJ -o kpt.tar.gz https://github.com/GoogleContainerTools/kpt/releases/download/v${KPT_VERSION}/kpt_darwin_amd64-${KPT_VERSION}.tar.gz > $OUTPUT_DIR/kptdownload 2>&1
    else
      echo -e "${RED}ERROR${NC}: Unknown OS $uname"
      exit 1
  fi
  tar -xvf kpt.tar.gz > $OUTPUT_DIR/kptdownload 2>&1
  mv kpt $BIN_DIR/kpt1.0.0
  echo -e "Downloading 1.0.0 kpt binary...${GREEN}SUCCESS${NC}"
  rm kpt.tar.gz LICENSES.txt lib.zip
  set +e
}

# buildKpt builds the kpt binary, storing it in the temporary directory.
# To check the stdout output of the build check $OUTPUT_DIR/kptbuild.
# stderr output will be output to the terminal.
function buildKpt {
    set -e
    if [ -z $BUILD_DEPS_AT_HEAD ]; then
	echo "checking go version"
	go version
	echo "Building kpt locally..."
	go build -o $BIN_DIR -v . > $OUTPUT_DIR/kptbuild 2>&1
	echo -e "Building kpt locally...${GREEN}SUCCESS${NC}"

    else
	echo "Building kpt using dependencies at HEAD..."
	echo
	# Clone kpt repository into kpt source directory
	KPT_SRC_DIR="${SRC_DIR}/github.com/GoogleContainerTools/kpt"
	mkdir -p $KPT_SRC_DIR
	echo "Downloading kpt repository at HEAD..."
	git clone https://github.com/GoogleContainerTools/kpt ${KPT_SRC_DIR} > ${OUTPUT_DIR}/kptbuild 2>&1
	echo -e "Downloading kpt repository at HEAD...${GREEN}SUCCESS${NC}"
	# Clone cli-utils repository into source directory
	CLI_UTILS_SRC_DIR="${SRC_DIR}/sigs.k8s.io/cli-utils"
	mkdir -p $CLI_UTILS_SRC_DIR
	echo "Downloading cli-utils repository at HEAD..."
	git clone https://github.com/kubernetes-sigs/cli-utils ${CLI_UTILS_SRC_DIR} > ${OUTPUT_DIR}/kptbuild 2>&1
	echo -e "Downloading cli-utils repository at HEAD...${GREEN}SUCCESS${NC}"
	# Clone kustomize respository into source directory
	KUSTOMIZE_SRC_DIR="${SRC_DIR}/sigs.k8s.io/kustomize"
	mkdir -p $KUSTOMIZE_SRC_DIR
	echo "Downloading kustomize repository at HEAD..."
	git clone https://github.com/kubernetes-sigs/kustomize ${KUSTOMIZE_SRC_DIR} > ${OUTPUT_DIR}/kptbuild 2>&1
	echo -e "Downloading kustomize repository at HEAD...${GREEN}SUCCESS${NC}"
	# Tell kpt to build using the locally downloaded dependencies
	echo "Updating kpt/go.mod to reference locally downloaded repositories..."
	echo -e "\n\nreplace sigs.k8s.io/cli-utils => ../../../sigs.k8s.io/cli-utils" >> ${KPT_SRC_DIR}/go.mod
	echo -e "replace sigs.k8s.io/kustomize/cmd/config => ../../../sigs.k8s.io/kustomize/cmd/config" >> ${KPT_SRC_DIR}/go.mod
	echo -e "replace sigs.k8s.io/kustomize/kyaml => ../../../sigs.k8s.io/kustomize/kyaml\n" >> ${KPT_SRC_DIR}/go.mod
	echo -e "Updating kpt/go.mod to reference locally downloaded repositories...${GREEN}SUCCESS${NC}"
	# Build kpt using the cloned directories
	export GOPATH=${TMP_DIR}
	echo "Building kpt..."
	(cd -- ${KPT_SRC_DIR} && go build -o $BIN_DIR -v . > ${OUTPUT_DIR}/kptbuild)
	echo -e "Building kpt...${GREEN}SUCCESS${NC}"
	echo
	echo -e "Building kpt using dependencies at HEAD...${GREEN}SUCCESS${NC}"
    fi
    set +e
}

# createTestSuite deletes then creates the kind cluster. We wait for the node
# to become ready before we run the tests.
function createTestSuite {
    set -e
    echo "Setting Up Test Suite..."
    echo
    # Create the k8s cluster
    echo "Deleting kind cluster..."
    kind delete cluster > /dev/null 2>&1
    echo -e "Deleting kind cluster...${GREEN}SUCCESS${NC}"
    echo "Creating kind cluster..."
    kind create cluster --image=kindest/node:v${K8S_VERSION} --wait=2m > $OUTPUT_DIR/k8sstartup 2>&1
    kubectl wait node/kind-control-plane --for condition=ready --timeout=2m
    echo -e "Creating kind cluster...${GREEN}SUCCESS${NC}"
    echo
    echo -e "Setting Up Test Suite...${GREEN}SUCCESS${NC}"
    echo
    set +e
}

function waitForDefaultServiceAccount {
    # Necessary to ensure default service account is created before pods.
    echo -n "Waiting for default service account..."
    echo -n ' '
    sp="/-\|"
    n=1
    until ((n >= 300)); do
	kubectl -n default get serviceaccount default -o name 2>&1 | tee $OUTPUT_DIR/status
	test 1 == $(grep "serviceaccount/default" $OUTPUT_DIR/status | wc -l)
	if [ $? == 0 ]; then
	    echo
	    break
	fi
	printf "\b${sp:n++%${#sp}:1}"
	sleep 0.2
    done
    ((n < 300))
    echo "Waiting for default service account...CREATED"
    echo
}

# assertKptLiveApplyEquals checks that the STDIN equals the content of the
# $OUTPUT_DIR/status file, after filtering and sorting.
function assertKptLiveApplyEquals {
  local expected="$(cat | processKptLiveOutput)"
  local received="$(cat "$OUTPUT_DIR/status" | processKptLiveOutput)"
  local diff_result="$( diff -u <(echo -e "${expected}") <(echo -e "${received}") --label "expected" --label "received")"
  if [[ -z "${diff_result}" ]]; then
    echo -n '.'
  else
    echo -n 'E'
    echo -e "error: expected output ():\n${diff_result}" >> "$OUTPUT_DIR/errors"
    HAS_TEST_FAILURE=1
  fi
}

function processKptLiveOutput {
    trimTrailingNewlines | \
    filterReconcilePending | \
    filterUnknownFieldsWarning | \
    sortReconcileEvents | \
    sortActuationEvents
}

function trimTrailingNewlines {
    sed -e :a -e '/^\n*$/{$d;N;ba' -e '}'
}

function filterReconcilePending {
  grep -v " reconcile pending$" || true
}

function filterUnknownFieldsWarning {
  grep -v " unknown field" || true
}

# sortReconcileEvents sorts reconcile events: successful > failed.
# Not sorted: skipped (always first) & timeout (always last).
function sortReconcileEvents {
  local input="$(cat)" # read full input before sorting
  local status=""
  local successfulBuffer=""
  local failedBuffer=""
  local pattern="^.* reconcile (successful|failed).*$"
  while IFS="" read -r line; do
    if [[ "${line}" =~ ${pattern} ]]; then
      # match - add line to buffer
      status="${BASH_REMATCH[1]}"
      if [[ "${status}" == "successful" ]]; then
        successfulBuffer+="${line}\n"
      elif [[ "${status}" == "failed" ]]; then
        failedBuffer+="${line}\n"
      else
        echo "ERROR: Unexpected reconcile status: ${status}"
        return
      fi
    else
      # no match - dump buffers & print line
      if [[ -n "${successfulBuffer}" ]]; then
        echo -en "${successfulBuffer}" | sort
        successfulBuffer=""
      fi
      if [[ -n "${failedBuffer}" ]]; then
        echo -en "${failedBuffer}" | sort
        failedBuffer=""
      fi
      echo "${line}"
    fi
  done <<< "${input}"
  # end of input - dump buffers
  if [[ -n "${successfulBuffer}" ]]; then
    echo -en "${successfulBuffer}" | sort
    successfulBuffer=""
  fi
  if [[ -n "${failedBuffer}" ]]; then
    echo -en "${failedBuffer}" | sort
    failedBuffer=""
  fi
}

# sortActuationEvents sorts apply/prune/delete events: skipped > successful > failed.
# Only events with the same actuation type and resource type are sorted.
function sortActuationEvents {
  local input="$(cat)" # read full input before sorting
  local status=""
  local resource=""
  local actuation=""
  local skippedBuffer=""
  local successfulBuffer=""
  local failedBuffer=""
  local pattern="^(.*)/(.*) (apply|prune|delete) (skipped|successful|failed).*$"
  while IFS="" read -r line; do
    if [[ "${line}" =~ ${pattern} ]]; then
      # match
      if [[ "${resource}" != "${BASH_REMATCH[1]}" ]] || [[ "${actuation}" != "${BASH_REMATCH[3]}" ]]; then
        # different resource/actuation - dump buffers & update resource/actuation
        if [[ -n "${skippedBuffer}" ]]; then
          echo -en "${skippedBuffer}" | sort
          skippedBuffer=""
        fi
        if [[ -n "${successfulBuffer}" ]]; then
          echo -en "${successfulBuffer}" | sort
          successfulBuffer=""
        fi
        if [[ -n "${failedBuffer}" ]]; then
          echo -en "${failedBuffer}" | sort
          failedBuffer=""
        fi
        resource="${BASH_REMATCH[1]}"
        actuation="${BASH_REMATCH[3]}"
      fi

      # add line to buffer
      status="${BASH_REMATCH[4]}"
      if [[ "${status}" == "skipped" ]]; then
        skippedBuffer+="${line}\n"
      elif [[ "${status}" == "successful" ]]; then
        successfulBuffer+="${line}\n"
      elif [[ "${status}" == "failed" ]]; then
        failedBuffer+="${line}\n"
      else
        echo "ERROR: Unexpected ${actuation} status: ${status}"
        return
      fi
    else
      # no match - dump buffers & print line
      if [[ -n "${skippedBuffer}" ]]; then
        echo -en "${skippedBuffer}" | sort
        skippedBuffer=""
      fi
      if [[ -n "${successfulBuffer}" ]]; then
        echo -en "${successfulBuffer}" | sort
        successfulBuffer=""
      fi
      if [[ -n "${failedBuffer}" ]]; then
        echo -en "${failedBuffer}" | sort
        failedBuffer=""
      fi
      echo "${line}"
      resource=""
      actuation=""
    fi
  done <<< "${input}"
  # end of input - dump buffers
  if [[ -n "${skippedBuffer}" ]]; then
    echo -en "${skippedBuffer}" | sort
    skippedBuffer=""
  fi
  if [[ -n "${successfulBuffer}" ]]; then
    echo -en "${successfulBuffer}" | sort
    successfulBuffer=""
  fi
  if [[ -n "${failedBuffer}" ]]; then
    echo -en "${failedBuffer}" | sort
    failedBuffer=""
  fi
}

# assertContains checks that the passed string is a substring of
# the $OUTPUT_DIR/status file.
function assertContains {
  local test_arg="$@"
  test 1 == \
  $(grep "$test_arg" $OUTPUT_DIR/status | wc -l); \
  if [ $? == 0 ]; then
      echo -n '.'
  else
      echo -n 'E'
      if [ ! -f $OUTPUT_DIR/errors ]; then
	  touch $OUTPUT_DIR/errors
      fi
      echo "error: expected missing text \"${test_arg}\"" >> $OUTPUT_DIR/errors
      HAS_TEST_FAILURE=1
  fi
}

# assertNotContains checks that the passed string is NOT a substring of
# the $OUTPUT_DIR/status file.
function assertNotContains {
  local test_arg="$@"
  test 0 == \
  $(grep "$test_arg" $OUTPUT_DIR/status | wc -l); \
  if [ $? == 0 ]; then
      echo -n '.'
  else
      echo -n 'E'
      if [ ! -f $OUTPUT_DIR/errors ]; then
	  touch $OUTPUT_DIR/errors
      fi
      echo "error: unexpected text \"${test_arg}\" found" >> $OUTPUT_DIR/errors
      HAS_TEST_FAILURE=1
  fi
}

# assertCMInventory checks that a ConfigMap inventory object exists in
# the passed namespace with the passed number of inventory items.
# Assumes the inventory object name begins with "inventory-".
function assertCMInventory {
    local ns=$1
    local numInv=$2
    
    echo "kubectl get cm -n $ns --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers"
    inv=$(kubectl get cm -n $ns --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers)
    echo $inv | awk '{print $1}' > $OUTPUT_DIR/invname
    echo $inv | awk '{print $2}' > $OUTPUT_DIR/numinv

    test 1 == $(grep "inventory-" $OUTPUT_DIR/invname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	if [ ! -f $OUTPUT_DIR/errors ]; then
	    touch $OUTPUT_DIR/errors
	fi
	echo "error: expected missing ConfigMap inventory object in ${ns} namespace" >> $OUTPUT_DIR/errors
    HAS_TEST_FAILURE=1
    fi

    test 1 == $(grep $numInv $OUTPUT_DIR/numinv | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	if [ ! -f $OUTPUT_DIR/errors ]; then
	    touch $OUTPUT_DIR/errors
	fi
	echo "error: expected ConfigMap inventory to have $numInv inventory items" >> $OUTPUT_DIR/errors
    HAS_TEST_FAILURE=1
    fi
}

# assertRGInventory checks that a ResourceGroup inventory object exists
# in the passed namespace. Assumes the inventory object name begins
# with "inventory-".
function assertRGInventory {
    local ns=$1
    
    echo "kubectl get resourcegroups.kpt.dev -n $ns --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers | awk '{print $1}'"
    kubectl get resourcegroups.kpt.dev -n $ns --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers | awk '{print $1}' > $OUTPUT_DIR/invname

    test 1 == $(grep "inventory-" $OUTPUT_DIR/invname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	if [ ! -f $OUTPUT_DIR/errors ]; then
	    touch $OUTPUT_DIR/errors
	fi
	echo "error: expected missing ResourceGroup inventory in ${ns} namespace" >> $OUTPUT_DIR/errors
    HAS_TEST_FAILURE=1
    fi
}

# assertPodExists checks that a pod with the passed podName and passed
# namespace exists in the cluster.
TIMEOUT_SECS=30
function assertPodExists {
    local podName=$1
    local namespace=$2

    echo "kubectl wait --for=condition=Ready -n $namespace pod/$podName --timeout=${TIMEOUT_SECS}s"
    kubectl wait --for=condition=Ready -n $namespace pod/$podName --timeout=${TIMEOUT_SECS}s > /dev/null 2>&1
    echo "kubectl get po -n $namespace $podName -o name | awk '{print $1}'"
    kubectl get po -n $namespace $podName -o name | awk '{print $1}' > $OUTPUT_DIR/podname

    test 1 == $(grep $podName $OUTPUT_DIR/podname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	if [ ! -f $OUTPUT_DIR/errors ]; then
	    touch $OUTPUT_DIR/errors
	fi
	echo "error: expected missing pod $namespace/$podName in ${namespace} namespace" >> $OUTPUT_DIR/errors
    HAS_TEST_FAILURE=1
    fi
}

# assertPodNotExists checks that a pod with the passed podName and passed
# namespace DOES NOT exist in the cluster. Waits 20 seconds for pod
# termination if pod has not finished deleting.
function assertPodNotExists {
    local podName=$1
    local namespace=$2

    echo "kubectl wait --for=delete -n $namespace pod/$podName --timeout=${TIMEOUT_SECS}s"
    kubectl wait --for=delete -n $namespace pod/$podName --timeout=${TIMEOUT_SECS}s > /dev/null 2>&1
    echo "kubectl get po -n $namespace $podName -o name"
    kubectl get po -n $namespace $podName -o name > $OUTPUT_DIR/podname 2>&1
    
    test 1 == $(grep "(NotFound)" $OUTPUT_DIR/podname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	if [ ! -f $OUTPUT_DIR/errors ]; then
	    touch $OUTPUT_DIR/errors
	fi
	echo "error: unexpected pod $namespace/$podName found in ${namespace} namespace" >> $OUTPUT_DIR/errors
    HAS_TEST_FAILURE=1
    fi
}

# printResult prints the results of the previous assert statements
function printResult {
    if [ -f $OUTPUT_DIR/errors ]; then
	echo -e "${RED}ERROR${NC}"
	cat $OUTPUT_DIR/errors
	echo
	rm -f $OUTPUT_DIR/errors
    else
	echo -e "${GREEN}SUCCESS${NC}"
    fi
    echo
}

# wait sleeps for the passed number of seconds.
function wait {
    local numSecs=$1

    sleep $numSecs
}

SEMVER_PATTERN="v(.*)\.(.*)\.(.*)"

function kubeServerVersion {
  kubectl version --output=json | jq -r '.serverVersion.gitVersion'
}

function kubeServerMinorVersion {
  if [[ "$(kubeServerVersion)" =~ ${SEMVER_PATTERN} ]]; then
    echo "${BASH_REMATCH[2]}"
  else
    return 1
  fi
}

###########################################################################
#  Main
###########################################################################

# Setup temporary directory for src, bin, and output.
TMP_DIR=$(mktemp -d -t kpt-e2e-XXXXXXXXXX)
SRC_DIR="${TMP_DIR}/src"
mkdir -p $SRC_DIR
BIN_DIR="${TMP_DIR}/bin"
mkdir -p ${BIN_DIR}
OUTPUT_DIR="${TMP_DIR}/output"
mkdir -p $OUTPUT_DIR

# Build the kpt binary and copy it to the temp dir. If BUILD_DEPS_AT_HEAD
# is set, then copy the kpt repository AND dependency directories into
# TMP_DIR and build from there.
echo "kpt end-to-end test"
echo
echo "Kubernetes Version: ${K8S_VERSION}"
echo "Temp Dir: $TMP_DIR"
echo

buildKpt

downloadPreviousKpt
downloadKpt1.0

echo
set +e                          # Do not stop the test for errors

#############################################################
#  Tests without initial ResourceGroup inventory installation
#############################################################

createTestSuite
waitForDefaultServiceAccount

# Basic init as setup for follow-on tests
# Test: Apply dry-run without ResourceGroup CRD installation fails
echo "[ResourceGroup] Testing initial apply dry-run without ResourceGroup inventory CRD"
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1a/
echo "kpt live init --quiet e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live init --quiet e2e/live/testdata/rg-test-case-1a
echo "kpt live apply --dry-run e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live apply --dry-run e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "Error: The ResourceGroup CRD was not found in the cluster. Please install it either by using the '--install-resource-group' flag or the 'kpt live install-resource-group' command."
printResult

# Test: Apply installs ResourceGroup CRD
echo "[ResourceGroup] Testing create inventory CRD before basic apply"
echo "kpt live apply e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live apply e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
# The ResourceGroup inventory CRD is automatically installed on the initial apply.
assertKptLiveApplyEquals << EOF
installing inventory ResourceGroup CRD.
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/rg-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertRGInventory "rg-test-namespace" "4"
printResult

# Apply again, but the ResourceGroup CRD is not re-installed.
echo "kpt live apply e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live apply e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertNotContains "installing inventory ResourceGroup CRD"  # Not applied again
assertKptLiveApplyEquals << EOF
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/rg-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Cleanup by resetting Kptfile and deleting resourcegroup file
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1a/
rm e2e/live/testdata/rg-test-case-1a/resourcegroup.yaml

###########################################################
#  Tests operations with ResourceGroup inventory CRD
###########################################################

createTestSuite
waitForDefaultServiceAccount

# Test: Installing ResourceGroup CRD
echo "Installing ResourceGroup CRD"
# First, check that the ResourceGroup CRD does NOT exist
echo "kubectl get resourcegroups.kpt.dev"
kubectl get resourcegroups.kpt.dev 2>&1 | tee $OUTPUT_DIR/status
assertContains "error: the server doesn't have a resource type \"resourcegroups\""
# Next, add the ResourceGroup CRD
echo "kpt live install-resource-group"
${BIN_DIR}/kpt live install-resource-group 2>&1 | tee $OUTPUT_DIR/status
assertContains "installing inventory ResourceGroup CRD...success"
echo "kubectl get resourcegroups.kpt.dev"
kubectl get resourcegroups.kpt.dev 2>&1 | tee $OUTPUT_DIR/status
assertContains "No resources found"
# Add a simple ResourceGroup custom resource, and verify it exists in the cluster.
echo "kubectl apply -f e2e/live/testdata/install-rg-crd/example-resource-group.yaml"
kubectl apply -f e2e/live/testdata/install-rg-crd/example-resource-group.yaml 2>&1 | tee $OUTPUT_DIR/status
assertContains "resourcegroup.kpt.dev/example-inventory created"
echo "kubectl get resourcegroups.kpt.dev --no-headers"
kubectl get resourcegroups.kpt.dev --no-headers 2>&1 | tee $OUTPUT_DIR/status
assertContains "example-inventory"
# Finally, add the ResourceGroup CRD again, and check it says it already exists.
echo "kpt live install-resource-group"
${BIN_DIR}/kpt live install-resource-group 2>&1 | tee $OUTPUT_DIR/status
assertContains "...success"
printResult

# Test: Basic Kptfile/ResourceGroup inititalizing inventory info
echo "Testing init for Kptfile/ResourceGroup"
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1a/
echo "kpt live init e2e/live/testdata/rg-test-case-1a"
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1a
${BIN_DIR}/kpt live init e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "initializing \"resourcegroup.yaml\" data (namespace: rg-test-namespace)...success"
# Difference in Kptfile should have inventory data
diff e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1a/Kptfile 2>&1 | tee $OUTPUT_DIR/status
assertNotContains "inventory:"
assertNotContains "namespace: rg-test-namespace"
assertNotContains "name: inventory-"
assertNotContains "inventoryID:"
# ResourceGroup file should contain inventory information
cat e2e/live/testdata/rg-test-case-1a/resourcegroup.yaml 2>&1 | tee $OUTPUT_DIR/status
assertContains "kind: ResourceGroup"
assertContains "namespace: rg-test-namespace"
assertContains "name: inventory-"
printResult

echo "Testing init Kptfile/ResourceGroup already initialized"
echo "kpt live init e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live init e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "initializing \"resourcegroup.yaml\" data (namespace: rg-test-namespace)...failed"
assertContains "Error: Inventory information has already been added to the package ResourceGroup object."
printResult

echo "Testing init force Kptfile/ResourceGroup"
echo "kpt live init --force --name inventory-18030002 e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live init --force --name inventory-18030002 e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "initializing \"resourcegroup.yaml\" data (namespace: rg-test-namespace)...success"
cat e2e/live/testdata/rg-test-case-1a/resourcegroup.yaml 2>&1 | tee $OUTPUT_DIR/status
assertContains "name: inventory-18030002"
printResult

echo "Testing init quiet Kptfile/ResourceGroup"
echo "kpt live init --quiet e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live init --quiet e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertNotContains "initializing resourcegroup"
printResult

# Test: Basic kpt live apply dry-run
# Apply run-run for "rg-test-case-1a" directory
echo "[ResourceGroup] Testing initial apply dry-run"
echo "kpt live apply --dry-run e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live apply --dry-run e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
Dry-run strategy: client
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
EOF
printResult

# Test: Basic kpt live apply
# Apply run for "rg-test-case-1a" directory
echo "[ResourceGroup] Testing basic apply"
echo "kpt live apply e2e/live/testdata/rg-test-case-1a"
${BIN_DIR}/kpt live apply e2e/live/testdata/rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
# The ResourceGroup CRD is already installed.
assertKptLiveApplyEquals << EOF
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/rg-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertRGInventory "rg-test-namespace" "4"
printResult

# Test: Basic kpt live apply on symlink
# Apply run for "rg-test-case-1a" directory
echo "[ResourceGroup] Testing basic apply on symlink"
rm -f link-to-rg-test-case-1a # Delete if exists
ln -s e2e/live/testdata/rg-test-case-1a/ link-to-rg-test-case-1a
trap "rm -f ${PWD}/link-to-rg-test-case-1a" EXIT
echo "kpt live apply link-to-rg-test-case-1a"
${BIN_DIR}/kpt live apply link-to-rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
# The ResourceGroup CRD is already installed.
assertNotContains "installing inventory ResourceGroup CRD"
assertKptLiveApplyEquals << EOF
[WARN] resolved symlink "link-to-rg-test-case-1a" to "e2e/live/testdata/rg-test-case-1a", please note that the symlinks within the package are ignored
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/rg-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertRGInventory "rg-test-namespace" "4"
printResult

# Test: Basic kpt live status on symlink
# Apply run for "rg-test-case-1a" directory
echo "[ResourceGroup] Testing basic status on symlink"
rm -f link-to-rg-test-case-1a # Delete if exists
ln -s e2e/live/testdata/rg-test-case-1a/ link-to-rg-test-case-1a
trap "rm -f ${PWD}/link-to-rg-test-case-1a" EXIT
echo "kpt live status link-to-rg-test-case-1a"
${BIN_DIR}/kpt live status link-to-rg-test-case-1a 2>&1 | tee $OUTPUT_DIR/status
# The ResourceGroup CRD is already installed.
assertNotContains "installing inventory ResourceGroup CRD"
assertKptLiveApplyEquals << EOF
[WARN] resolved symlink "link-to-rg-test-case-1a" to "e2e/live/testdata/rg-test-case-1a", please note that the symlinks within the package are ignored
inventory-18030002/namespace//rg-test-namespace is Current: Resource is current
inventory-18030002/pod/rg-test-namespace/pod-a is Current: Pod is Ready
inventory-18030002/pod/rg-test-namespace/pod-b is Current: Pod is Ready
inventory-18030002/pod/rg-test-namespace/pod-c is Current: Pod is Ready
EOF
printResult

# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertRGInventory "rg-test-namespace" "4"
printResult

rm -rf link-to-rg-test-case-1a

# Test: kpt live apply dry-run of with prune
# "rg-test-case-1b" directory is "rg-test-case-1a" directory with "pod-a" removed and "pod-d" added.
echo "[ResourceGroup] Testing basic apply dry-run"
cp -f e2e/live/testdata/rg-test-case-1a/Kptfile e2e/live/testdata/rg-test-case-1b/
echo "kpt live apply --dry-run e2e/live/testdata/rg-test-case-1b"
cp -f e2e/live/testdata/rg-test-case-1a/Kptfile e2e/live/testdata/rg-test-case-1b
cp -f e2e/live/testdata/rg-test-case-1a/resourcegroup.yaml e2e/live/testdata/rg-test-case-1b
${BIN_DIR}/kpt live apply --dry-run e2e/live/testdata/rg-test-case-1b 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
Dry-run strategy: client
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
apply phase started
pod/pod-b apply successful
pod/pod-c apply successful
pod/pod-d apply successful
apply phase finished
prune phase started
pod/pod-a prune successful
prune phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
prune result: 1 attempted, 1 successful, 0 skipped, 0 failed
EOF
printResult

# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertRGInventory "rg-test-namespace" "4"
assertPodExists "pod-a" "rg-test-namespace"
assertPodExists "pod-b" "rg-test-namespace"
assertPodExists "pod-c" "rg-test-namespace"
printResult

# Test: Basic kpt live apply/prune
# "rg-test-case-1b" directory is "rg-test-case-1a" directory with "pod-a" removed and "pod-d" added.
echo "[ResourceGroup] Testing basic prune"
echo "kpt live apply e2e/live/testdata/rg-test-case-1b"
${BIN_DIR}/kpt live apply e2e/live/testdata/rg-test-case-1b 2>&1 | tee $OUTPUT_DIR/status
assertNotContains "installing inventory ResourceGroup CRD"  # CRD already installed
assertKptLiveApplyEquals << EOF
inventory update started
inventory update finished
apply phase started
namespace/rg-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/rg-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-b apply successful
pod/pod-c apply successful
pod/pod-d apply successful
apply phase finished
reconcile phase started
pod/pod-b reconcile successful
pod/pod-c reconcile successful
pod/pod-d reconcile successful
reconcile phase finished
prune phase started
pod/pod-a prune successful
prune phase finished
reconcile phase started
pod/pod-a reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
prune result: 1 attempted, 1 successful, 0 skipped, 0 failed
reconcile result: 5 attempted, 5 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertRGInventory "rg-test-namespace" "4"
assertPodExists "pod-b" "rg-test-namespace"
assertPodExists "pod-c" "rg-test-namespace"
assertPodExists "pod-d" "rg-test-namespace"
assertPodNotExists "pod-a" "rg-test-namespace"
printResult

# Basic kpt live destroy --dry-run
echo "[ResourceGroup] Testing basic destroy dry-run"
echo "kpt live destroy --dry-run e2e/live/testdata/rg-test-case-1b"
${BIN_DIR}/kpt live destroy --dry-run e2e/live/testdata/rg-test-case-1b 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
Dry-run strategy: client
delete phase started
pod/pod-d delete successful
pod/pod-c delete successful
pod/pod-b delete successful
delete phase finished
delete phase started
namespace/rg-test-namespace delete successful
delete phase finished
inventory update started
inventory update finished
delete result: 4 attempted, 4 successful, 0 skipped, 0 failed
EOF
printResult

# Validate resources NOT DESTROYED in the cluster
assertPodExists "pod-b" "rg-test-namespace"
assertPodExists "pod-c" "rg-test-namespace"
assertPodExists "pod-d" "rg-test-namespace"
printResult

# Test: Basic kpt live destroy
# "rg-test-case-1b" directory is "rg-test-case-1a" directory with "pod-a" removed and "pod-d" added.
echo "[ResourceGroup] Testing basic destroy"
echo "kpt live destroy e2e/live/testdata/rg-test-case-1b"
${BIN_DIR}/kpt live destroy e2e/live/testdata/rg-test-case-1b 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
delete phase started
pod/pod-d delete successful
pod/pod-c delete successful
pod/pod-b delete successful
delete phase finished
reconcile phase started
pod/pod-c reconcile successful
pod/pod-b reconcile successful
pod/pod-d reconcile successful
reconcile phase finished
delete phase started
namespace/rg-test-namespace delete successful
delete phase finished
reconcile phase started
namespace/rg-test-namespace reconcile successful
reconcile phase finished
inventory update started
inventory update finished
delete result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Validate resources NOT in the cluster
assertPodNotExists "pod-b" "rg-test-namespace"
assertPodNotExists "pod-c" "rg-test-namespace"
assertPodNotExists "pod-d" "rg-test-namespace"
printResult

# Test: Basic kpt live apply/status/destroy from stdin
# 
echo "Testing apply/status/destroy from stdin"
echo "cat e2e/live/testdata/stdin-test/pods.yaml | kpt live apply -"
cat e2e/live/testdata/stdin-test/pods.yaml | ${BIN_DIR}/kpt live apply - 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
inventory update started
inventory update finished
apply phase started
namespace/stdin-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/stdin-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

echo "cat e2e/live/testdata/stdin-test/pods.yaml | kpt live status -"
cat e2e/live/testdata/stdin-test/pods.yaml | ${BIN_DIR}/kpt live status - 2>&1 | tee $OUTPUT_DIR/status
assertContains "inventory-18030002/namespace//stdin-test-namespace is Current: Resource is current"
assertContains "inventory-18030002/pod/stdin-test-namespace/pod-a is Current: Pod is Ready"
assertContains "inventory-18030002/pod/stdin-test-namespace/pod-b is Current: Pod is Ready"
assertContains "inventory-18030002/pod/stdin-test-namespace/pod-c is Current: Pod is Ready"
printResult

echo "cat e2e/live/testdata/stdin-test/pods.yaml | kpt live destroy -"
cat e2e/live/testdata/stdin-test/pods.yaml | ${BIN_DIR}/kpt live destroy - 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
delete phase started
pod/pod-c delete successful
pod/pod-b delete successful
pod/pod-a delete successful
delete phase finished
reconcile phase started
pod/pod-c reconcile successful
pod/pod-b reconcile successful
pod/pod-a reconcile successful
reconcile phase finished
delete phase started
namespace/stdin-test-namespace delete successful
delete phase finished
reconcile phase started
namespace/stdin-test-namespace reconcile successful
reconcile phase finished
inventory update started
inventory update finished
delete result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Test: kpt live apply continue-on-error
echo "[ResourceGroup] Testing continue-on-error"
echo "kpt live apply e2e/live/testdata/continue-on-error"
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/continue-on-error
${BIN_DIR}/kpt live init e2e/live/testdata/continue-on-error 2>&1 | tee $OUTPUT_DIR/status
diff e2e/live/testdata/Kptfile e2e/live/testdata/continue-on-error/resourcegroup.yaml 2>&1 | tee $OUTPUT_DIR/status
assertContains "namespace: continue-err-namespace"
printResult

echo "kpt live apply e2e/live/testdata/continue-on-error"
${BIN_DIR}/kpt live apply e2e/live/testdata/continue-on-error 2>&1 | tee $OUTPUT_DIR/status

if [[ "$(kubeServerMinorVersion)" -ge 20 ]]; then # >= 1.20.x
  # https://github.com/kubernetes/kubernetes/blob/v1.20.0/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L199
  RFC1123_ERROR="a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"
else # < 1.20.x
  # https://github.com/kubernetes/kubernetes/blob/v1.19.0/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L199
  RFC1123_ERROR="a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"
fi

assertKptLiveApplyEquals << EOF
inventory update started
inventory update finished
apply phase started
namespace/continue-err-namespace apply successful
apply phase finished
reconcile phase started
namespace/continue-err-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-B apply failed: error when creating "pod-b.yaml": Pod "pod-B" is invalid: metadata.name: Invalid value: "pod-B": ${RFC1123_ERROR} (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
apply phase finished
reconcile phase started
pod/pod-B reconcile skipped
pod/pod-a reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 3 attempted, 2 successful, 0 skipped, 1 failed
reconcile result: 3 attempted, 2 successful, 1 skipped, 0 failed, 0 timed out
EOF
printResult

assertRGInventory "continue-err-namespace" "2"
assertPodExists "pod-a" "continue-err-namespace"
assertPodNotExists "pod-B" "continue-err-namespace"
printResult

# Test: RBAC error applying a resource
echo "Testing RBAC error during apply"
# Setup: create a service account and bind a Role to it so it has administrative
# privileges on the "test" namespace, but no permissions on the default
# namespace.
echo "kubectl apply -f e2e/live/testdata/rbac-error-step-1"
kubectl apply -f e2e/live/testdata/rbac-error-step-1 2>&1 | tee $OUTPUT_DIR/status
assertContains "namespace/rbac-error created"
assertContains "rolebinding.rbac.authorization.k8s.io/admin created"
assertContains "serviceaccount/user created"
printResult
wait 2

# Setup: use the service account just created. It does not have permissions
# on the default namespace, so it will give a permissions error on apply
# for anything attempted to apply to the default namespace.
echo "kubectl apply -f e2e/live/testdata/rbac-error-step-2"
kubectl apply -f e2e/live/testdata/rbac-error-step-2 2>&1 | tee $OUTPUT_DIR/status
assertContains "secret/user-credentials created"
wait 2
SECRET_NAME="user-credentials"
echo "kubectl get secrets -ojsonpath='{.data.token}' "${SECRET_NAME}" | base64 -d"
SECRET_TOKEN="$(kubectl get secrets -ojsonpath='{.data.token}' "${SECRET_NAME}" | base64 -d)"
echo "kubectl config set-credentials user --token \"<REDACTED>\""
kubectl config set-credentials user --token "${SECRET_TOKEN}" 2>&1 | tee $OUTPUT_DIR/status
echo "kubectl config set-context kind-kind:user --cluster=kind-kind"
kubectl config set-context kind-kind:user --cluster=kind-kind --user=user 2>&1 | tee $OUTPUT_DIR/status
echo "kubectl config use-context kind-kind:user"
kubectl config use-context kind-kind:user 2>&1 | tee $OUTPUT_DIR/status
printResult
wait 2

# Attempt to apply two ConfigMaps: one in the default namespace (fails), and one
# in the "rbac-error" namespace (succeeds).
echo "kpt live apply --install-resource-group=false e2e/live/testdata/rbac-error-step-3"
${BIN_DIR}/kpt live apply --install-resource-group=false e2e/live/testdata/rbac-error-step-3 2>&1 | tee $OUTPUT_DIR/status
assertNotContains "installing inventory ResourceGroup CRD"  # CRD already installed
assertContains 'error: polling for status failed: failed to list /v1, Kind=ConfigMap: configmaps is forbidden: User "system:serviceaccount:default:user" cannot list resource "configmaps" in API group "" at the cluster scope'
printResult

# No inventory expected - permission error causes early exit
echo "kubectl get resourcegroups.kpt.dev -n 'rbac-error' --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers"
kubectl get resourcegroups.kpt.dev -n 'rbac-error' --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers 2>&1 | tee $OUTPUT_DIR/status
assertContains "No resources found"
printResult

###########################################################
#  Test Migrate from ConfigMap inventory to ResourceGroup
###########################################################

createTestSuite
waitForDefaultServiceAccount

# Setup: kpt live apply ConfigMap inventory
# Applies resources in "migrate-case-1a" directory.
echo "Testing kpt live apply with ConfigMap inventory"
# Prerequisite: set up the ConfigMap inventory file
# Copy Kptfile into "migrate-case-1a" WITHOUT inventory information. This ensures
# the apply uses the ConfigMap inventory-template.yaml during the apply.
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/migrate-case-1a/
cp -f e2e/live/testdata/template-rg-namespace.yaml e2e/live/testdata/migrate-case-1a/inventory-template.yaml
echo "previouskpt live apply e2e/live/testdata/migrate-case-1a"
${BIN_DIR}/previouskpt live apply e2e/live/testdata/migrate-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "namespace/test-rg-namespace unchanged"
assertContains "pod/pod-a created"
assertContains "pod/pod-b created"
assertContains "pod/pod-c created"
assertContains "4 resource(s) applied. 3 created, 1 unchanged, 0 configured, 0 failed"
assertContains "0 resource(s) pruned, 0 skipped, 0 failed"
printResult

# Validate resources in the cluster
assertCMInventory "test-rg-namespace" "4"
assertPodExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
printResult

# Test: kpt live migrate from ConfigMap to ResourceGroup inventory
# Migrates resources in "migrate-case-1a" directory.
echo "Testing migrate dry-run from ConfigMap to ResourceGroup inventory"
# Run migrate dry-run and verify that the migrate did not actually happen
echo "kpt live migrate --dry-run e2e/live/testdata/migrate-case-1a"
${BIN_DIR}/kpt live migrate --dry-run e2e/live/testdata/migrate-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "retrieve the current ConfigMap inventory...success (inventory-id:"
assertContains "creating ResourceGroup object file...success"
assertContains "retrieve ConfigMap inventory objs...success (4 inventory objects)"
assertContains "migrate inventory to ResourceGroup...success"
assertContains "deleting old ConfigMap inventory object...success"
assertContains "deleting inventory template file"
assertContains "inventory migration...success"
printResult

# Migrate did not actually happen in dry-run, so ConfigMap inventory still exists
assertCMInventory "test-rg-namespace" "4"
printResult

# Now actually run the migrate and verify the new ResourceGroup inventory exists
echo "Testing migrate from ConfigMap to ResourceGroup inventory"
# Prerequisite: set up the ConfigMap inventory file
cp -f e2e/live/testdata/template-rg-namespace.yaml e2e/live/testdata/migrate-case-1a/inventory-template.yaml
echo "kpt live migrate e2e/live/testdata/migrate-case-1a"
${BIN_DIR}/kpt live migrate e2e/live/testdata/migrate-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "retrieve the current ConfigMap inventory...success (inventory-id:"
assertContains "creating ResourceGroup object file...success"
assertContains "retrieve ConfigMap inventory objs...success (4 inventory objects)"
assertContains "migrate inventory to ResourceGroup...success"
assertContains "deleting old ConfigMap inventory object...success"
assertContains "deleting inventory template file"
assertContains "inventory migration...success"
printResult

# Validate resources in the cluster
assertPodExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
assertRGInventory "test-rg-namespace"
printResult

# Run it again, and validate the output
${BIN_DIR}/kpt live migrate e2e/live/testdata/migrate-case-1a 2>&1 | tee $OUTPUT_DIR/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "retrieve the current ConfigMap inventory...no ConfigMap inventory...completed"
assertContains "inventory migration...success"
printResult

# Test: kpt live apply/prune
# "rg-test-case-1b" directory is "rg-test-case-1a" directory with "pod-a" removed and "pod-d" added.
echo "Testing apply/prune after migrate"
cp -f e2e/live/testdata/migrate-case-1a/Kptfile e2e/live/testdata/migrate-case-1b/
echo "kpt live apply e2e/live/testdata/migrate-case-1b"
cp -f e2e/live/testdata/migrate-case-1a/Kptfile e2e/live/testdata/migrate-case-1b
cp -f e2e/live/testdata/migrate-case-1a/resourcegroup.yaml e2e/live/testdata/migrate-case-1b
${BIN_DIR}/kpt live apply e2e/live/testdata/migrate-case-1b 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
inventory update started
inventory update finished
apply phase started
namespace/test-rg-namespace apply successful
apply phase finished
reconcile phase started
namespace/test-rg-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-b apply successful
pod/pod-c apply successful
pod/pod-d apply successful
apply phase finished
reconcile phase started
pod/pod-b reconcile successful
pod/pod-c reconcile successful
pod/pod-d reconcile successful
reconcile phase finished
prune phase started
pod/pod-a prune successful
prune phase finished
reconcile phase started
pod/pod-a reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
prune result: 1 attempted, 1 successful, 0 skipped, 0 failed
reconcile result: 5 attempted, 5 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Validate resources in the cluster
# ResourceGroup inventory with four inventory items.
assertRGInventory "test-rg-namespace" "4"
assertPodNotExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
assertPodExists "pod-d" "test-rg-namespace"
printResult

###########################################################
#  Test Update ResourceGroup CRD during apply
###########################################################

createTestSuite
waitForDefaultServiceAccount

# This test first applies a kpt package with kpt1.0.0,
# which uses the previous ResourceGroup CRD.
# Then it re-apply the same kpt package with the built kpt,
# which uses a new ResourceGroup CRD.
# It updates the ResourceGroup CRD and apply/prune works as expected.
echo "Testing apply with kpt 1.0.0 and re-apply/prune with built kpt"
echo "cat e2e/live/testdata/stdin-test/pods.yaml | kpt1.0.0 live apply -"
cat e2e/live/testdata/stdin-test/pods.yaml | ${BIN_DIR}/kpt1.0.0 live apply - 2>&1 | tee $OUTPUT_DIR/status
assertContains "pod/pod-a created"
assertContains "pod/pod-b created"
assertContains "pod/pod-c created"
assertContains "4 resource(s) applied. 3 created, 1 unchanged, 0 configured, 0 failed"
printResult

echo "cat e2e/live/testdata/stdin-test/pods.yaml | kpt live apply -"
cat e2e/live/testdata/stdin-test/pods.yaml | ${BIN_DIR}/kpt live apply - 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
installing inventory ResourceGroup CRD.
inventory update started
inventory update finished
apply phase started
namespace/stdin-test-namespace apply successful
apply phase finished
reconcile phase started
namespace/stdin-test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

echo "cat e2e/live/testdata/stdin-test/pods.yaml | kpt live destroy -"
cat e2e/live/testdata/stdin-test/pods.yaml | ${BIN_DIR}/kpt live destroy - 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
delete phase started
pod/pod-c delete successful
pod/pod-b delete successful
pod/pod-a delete successful
delete phase finished
reconcile phase started
pod/pod-b reconcile successful
pod/pod-c reconcile successful
pod/pod-a reconcile successful
reconcile phase finished
delete phase started
namespace/stdin-test-namespace delete successful
delete phase finished
reconcile phase started
namespace/stdin-test-namespace reconcile successful
reconcile phase finished
inventory update started
inventory update finished
delete result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
printResult

# Test: don't have permission to update the ResourceGroup CRD
# should see an error message
echo "Testing updating ResourceGroup CRD during apply"
echo "Apply the previous ResourceGroup CRD"
echo "kpt1.0.0 live install-resource-group"
${BIN_DIR}/kpt1.0.0 live install-resource-group 2>&1 | tee $OUTPUT_DIR/status
# Setup: create a service account and bind a Role to it so it has administrative
# privileges on the "test" namespace, but no permissions to Get or Update CRD.
echo "kubectl apply -f e2e/live/testdata/rbac-error-step-1"
kubectl apply -f e2e/live/testdata/rbac-error-step-1 2>&1 | tee $OUTPUT_DIR/status
assertContains "namespace/rbac-error created"
assertContains "rolebinding.rbac.authorization.k8s.io/admin created"
assertContains "serviceaccount/user created"
wait 2

# Setup: use the service account just created. It does not have permissions
# on the default namespace, so it will give a permissions error on apply
# for anything attempted to apply to the default namespace.
echo "kubectl apply -f e2e/live/testdata/rbac-error-step-2"
kubectl apply -f e2e/live/testdata/rbac-error-step-2 2>&1 | tee $OUTPUT_DIR/status
assertContains "secret/user-credentials created"
wait 2
SECRET_NAME="user-credentials"
echo "kubectl get secrets -ojsonpath='{.data.token}' "${SECRET_NAME}" | base64 -d"
SECRET_TOKEN="$(kubectl get secrets -ojsonpath='{.data.token}' "${SECRET_NAME}" | base64 -d)"
echo "kubectl config set-credentials user --token \"<REDACTED>\""
kubectl config set-credentials user --token "${SECRET_TOKEN}" 2>&1 | tee $OUTPUT_DIR/status
echo "kubectl config set-context kind-kind:user --cluster=kind-kind --user=user"
kubectl config set-context kind-kind:user --cluster=kind-kind --user=user 2>&1 | tee $OUTPUT_DIR/status
echo "kubectl config use-context kind-kind:user"
kubectl config use-context kind-kind:user 2>&1 | tee $OUTPUT_DIR/status
wait 2

# Attempt to apply a kpt package. It fails with an error message.
echo "kpt live apply e2e/live/testdata/rbac-error-step-3"
${BIN_DIR}/kpt live apply e2e/live/testdata/rbac-error-step-3 2>&1 | tee $OUTPUT_DIR/status
assertContains "error: Type ResourceGroup CRD needs update."
printResult

###########################################################
#  Test Migrate from Kptfile inventory to ResourceGroup
###########################################################

createTestSuite
waitForDefaultServiceAccount

# Setup: kpt live apply ConfigMap inventory
# Applies resources in "migrate-case-2a" directory.
echo "Initialize Kptfile with inventory for migration to ResourceGroup tests"
echo "kpt live init e2e/live/testdata/migrate-case-2a"
${BIN_DIR}/kpt1.0.0 live init e2e/live/testdata/migrate-case-2a 2>&1 | tee $OUTPUT_DIR/status
assertContains "initializing Kptfile inventory info (namespace: test-namespace)...success"
printResult

echo "Testing kpt live apply with Kptfile inventory"
echo "kpt live apply e2e/live/testdata/migrate-case-2a"
${BIN_DIR}/kpt1.0.0 live apply e2e/live/testdata/migrate-case-2a 2>&1 | tee $OUTPUT_DIR/status
assertContains "installing inventory ResourceGroup CRD"
assertContains "namespace/test-namespace unchanged"
assertContains "pod/pod-a created"
assertContains "pod/pod-b created"
assertContains "pod/pod-c created"
assertContains "4 resource(s) applied. 3 created, 1 unchanged, 0 configured, 0 failed"
# Validate resources in the cluster
assertRGInventory "test-namespace" "4"
assertPodExists "pod-a" "test-namespace"
assertPodExists "pod-b" "test-namespace"
assertPodExists "pod-c" "test-namespace"
printResult

# Test: kpt live migrate from Kptfile inventory to ResourceGroup inventory
# Migrates resources in "migrate-case-2" directory.
echo "Testing migrate dry-run from Kptfile to ResourceGroup inventory"
echo "kpt live migrate --dry-run e2e/live/testdata/migrate-case-2a"
# Run migrate dry-run and verify that the migrate did not actually happen
${BIN_DIR}/kpt live migrate --dry-run e2e/live/testdata/migrate-case-2a 2>&1 | tee $OUTPUT_DIR/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "retrieve the current ConfigMap inventory...no ConfigMap inventory...completed"
assertContains "reading existing Kptfile...success"
assertContains "inventory migration...success"
printResult
# Ensure resourcegroup.yaml was not created
ls e2e/live/testdata/migrate-case-2a/resourcegroup.yaml 2>&1 | tee $OUTPUT_DIR/status
assertContains "ls: cannot access 'e2e/live/testdata/migrate-case-2a/resourcegroup.yaml': No such file or directory"
printResult

# Now actually run the migrate and verify the new ResourceGroup file exists
echo "Testing migrate from Kptfile to ResourceGroup inventory"
echo "kpt live migrate e2e/live/testdata/migrate-case-2a"
${BIN_DIR}/kpt live migrate e2e/live/testdata/migrate-case-2a 2>&1 | tee $OUTPUT_DIR/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "retrieve the current ConfigMap inventory...no ConfigMap inventory...completed"
assertContains "reading existing Kptfile...success"
assertContains "inventory migration...success"
# Validate resources in the cluster
assertPodExists "pod-a" "test-namespace"
assertPodExists "pod-b" "test-namespace"
assertPodExists "pod-c" "test-namespace"
assertRGInventory "test-namespace"
# ResourceGroup file should contain inventory information
cat e2e/live/testdata/migrate-case-2a/resourcegroup.yaml 2>&1 | tee $OUTPUT_DIR/status
assertContains "kind: ResourceGroup"
assertContains "namespace: test-namespace"
assertContains "name: inventory-"
printResult

# Run it again, and validate the output
${BIN_DIR}/kpt live migrate e2e/live/testdata/migrate-case-2a 2>&1 | tee $OUTPUT_DIR/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "retrieve the current ConfigMap inventory...no ConfigMap inventory...completed"
assertContains "reading existing Kptfile...inventory migration...success"
printResult

# Test: kpt live apply/prune
# "migrate-case-2b" directory is "migrate-case-2a" directory with "pod-a" removed and "pod-d" added.
echo "Testing apply/prune after migrate"
echo "kpt live apply e2e/live/testdata/migrate-case-2b"
cp -f e2e/live/testdata/migrate-case-2a/Kptfile e2e/live/testdata/migrate-case-2b
cp -f e2e/live/testdata/migrate-case-2a/resourcegroup.yaml e2e/live/testdata/migrate-case-2b
${BIN_DIR}/kpt live apply e2e/live/testdata/migrate-case-2b 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
installing inventory ResourceGroup CRD.
inventory update started
inventory update finished
apply phase started
namespace/test-namespace apply successful
apply phase finished
reconcile phase started
namespace/test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-b apply successful
pod/pod-c apply successful
pod/pod-d apply successful
apply phase finished
reconcile phase started
pod/pod-b reconcile successful
pod/pod-c reconcile successful
pod/pod-d reconcile pending
pod/pod-d reconcile successful
reconcile phase finished
prune phase started
pod/pod-a prune successful
prune phase finished
reconcile phase started
pod/pod-a reconcile pending
pod/pod-a reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
prune result: 1 attempted, 1 successful, 0 skipped, 0 failed
reconcile result: 5 attempted, 5 successful, 0 skipped, 0 failed, 0 timed out
EOF
# Validate resources in the cluster
# ResourceGroup inventory with four inventory items.
assertRGInventory "test-namespace" "4"
assertPodNotExists "pod-a" "test-namespace"
assertPodExists "pod-b" "test-namespace"
assertPodExists "pod-c" "test-namespace"
assertPodExists "pod-d" "test-namespace"
printResult

###########################################################
#  Test --rg-file flag on live init commands
###########################################################

createTestSuite
waitForDefaultServiceAccount

# Setup: kpt live init with custom resourcegroup file
# Applies resources in "test-case-1c" directory
echo "Testing kpt live init with custom ResourceGroup file"
echo "kpt live init --rg-file=custom-rg.yaml e2e/live/testdata/test-case-1c"
${BIN_DIR}/kpt live init --rg-file=custom-rg.yaml e2e/live/testdata/test-case-1c 2>&1 | tee $OUTPUT_DIR/status
assertContains "initializing \"custom-rg.yaml\" data (namespace: test-namespace)...success"
printResult
# Re-running live init should fail as ResourceGroup file already exists
${BIN_DIR}/kpt live init --rg-file=custom-rg.yaml e2e/live/testdata/test-case-1c 2>&1 | tee $OUTPUT_DIR/status
assertContains "initializing \"custom-rg.yaml\" data (namespace: test-namespace)...failed"
printResult

# Run: kpt live apply with custom resourcegroup file
echo "Testing kpt live apply with custom ResourceGroup filename"
echo "kpt live apply e2e/live/testdata/test-case-1c"
${BIN_DIR}/kpt live apply e2e/live/testdata/test-case-1c 2>&1 | tee $OUTPUT_DIR/status
cat $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
installing inventory ResourceGroup CRD.
inventory update started
inventory update finished
apply phase started
namespace/test-namespace apply successful
apply phase finished
reconcile phase started
namespace/test-namespace reconcile successful
reconcile phase finished
apply phase started
pod/pod-a apply successful
pod/pod-b apply successful
pod/pod-c apply successful
apply phase finished
reconcile phase started
pod/pod-a reconcile pending
pod/pod-b reconcile pending
pod/pod-c reconcile pending
pod/pod-a reconcile successful
pod/pod-b reconcile successful
pod/pod-c reconcile successful
reconcile phase finished
inventory update started
inventory update finished
apply result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
# Validate resources in the cluster
# ResourceGroup inventory with four inventory items.
assertRGInventory "test-namespace" "4"
assertPodExists "pod-a" "test-namespace"
assertPodExists "pod-b" "test-namespace"
assertPodExists "pod-c" "test-namespace"
printResult

echo "Testing live destroy with custom ResourceGroup filename"
echo "kpt live destroy --rg-file=custom-rg.yaml e2e/live/testdata/test-case-1c"
${BIN_DIR}/kpt live destroy e2e/live/testdata/test-case-1c 2>&1 | tee $OUTPUT_DIR/status
assertKptLiveApplyEquals << EOF
delete phase started
pod/pod-c delete successful
pod/pod-b delete successful
pod/pod-a delete successful
delete phase finished
reconcile phase started
pod/pod-c reconcile pending
pod/pod-b reconcile pending
pod/pod-a reconcile pending
pod/pod-b reconcile successful
pod/pod-c reconcile successful
pod/pod-a reconcile successful
reconcile phase finished
delete phase started
namespace/test-namespace delete successful
delete phase finished
reconcile phase started
namespace/test-namespace reconcile pending
namespace/test-namespace reconcile successful
reconcile phase finished
inventory update started
inventory update finished
delete result: 4 attempted, 4 successful, 0 skipped, 0 failed
reconcile result: 4 attempted, 4 successful, 0 skipped, 0 failed, 0 timed out
EOF
# Validate resources DESTROYED in the cluster
assertPodNotExists "pod-a" "test-namespace"
assertPodNotExists "pod-b" "test-namespace"
assertPodNotExists "pod-c" "test-namespace"
printResult

# Clean-up the k8s cluster
echo "Cleaning up cluster"
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1a/
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/rg-test-case-1b/
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/continue-on-error/
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/migrate-case-1a/
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/migrate-case-1b/
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/migrate-error/
kind delete cluster
echo -e "Cleaning up cluster...${GREEN}SUCCESS${NC}"

# Return error code if tests have failed
if [[ ${HAS_TEST_FAILURE} -gt 0 ]]; then
    echo -e "${RED}ERROR: E2E Tests Failed${NC}"
    exit ${HAS_TEST_FAILURE}
else
    echo -e "${GREEN}SUCCESS: E2E Tests Passed${NC}"
    exit 0
fi