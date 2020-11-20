#!/bin/bash
###########################################################################
# Copyright 2020 Google LLC
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
# Prerequisites (must be in $PATH):
#   kind - Kubernetes in Docker
#   kubectl - version of kubectl should be within +/- 1 version of cluster.
#     CHECK: kubectl version
#
###########################################################################

set +e

###########################################################################
#  Setup for test
###########################################################################

# Setup temporary directory for output.
BASE=$(mktemp -d -t ci-XXXXXXXXXX)
RESULT=$BASE/output
mkdir -p $RESULT

# Build the kpt binary and copy it to the temp dir.
echo "kpt end-to-end test"
echo
echo "Building kpt..."

go build -o $BASE -v .

echo "$BASE/kpt"
echo

# Create the k8s cluster
kind delete cluster
kind create cluster
echo
echo

# Necessary to ensure default service account is created before pods.
echo "Waiting for default service account..."
echo -n ' '
sp="/-\|"
n=1
until ((n >= 600)); do
    kubectl -n default get serviceaccount default -o name > $RESULT/status 2>&1
    test 1 == $(grep "serviceaccount/default" $RESULT/status | wc -l)
    if [ $? == 0 ]; then
	echo
	break
    fi
    printf "\b${sp:n++%${#sp}:1}"
    sleep 0.1
done
((n < 600))
echo "default service account created"
echo

###########################################################################
#  Helper functions
###########################################################################

# assertContains checks that the passed string is a substring of
# the $RESULT/status file.
ERROR=""
function assertContains {
  test 1 == \
  $(grep "$@" $RESULT/status | wc -l); \
  if [ $? == 0 ]; then
      echo -n '.'
  else
      echo -n 'E'
      ERROR+="ERROR: assertContains $@, but missing\n"
  fi
}

# assertCMInventory checks that a ConfigMap inventory object exists in
# the passed namespace with the passed number of inventory items.
# Assumes the inventory object name begins with "inventory-".
function assertCMInventory {
    local ns=$1
    local numInv=$2
    
    inv=$(kubectl get cm -n $ns --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers)
    echo $inv | awk '{print $1}' > $RESULT/invname
    echo $inv | awk '{print $2}' > $RESULT/numinv

    test 1 == $(grep "inventory-" $RESULT/invname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	ERROR+="ERROR: expected ConfigMap inventory to exist\n"
    fi

    test 1 == $(grep $numInv $RESULT/numinv | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	ERROR+="ERROR: expected ConfigMap inventory to have $numInv inventory items\n"
    fi
}

# assertRGInventory checks that a ResourceGroup inventory object exists
# in the passed namespace. Assumes the inventory object name begins
# with "inventory-".
function assertRGInventory {
    local ns=$1
    
    kubectl get resourcegroups.kpt.dev -n $ns --selector='cli-utils.sigs.k8s.io/inventory-id' --no-headers | awk '{print $1}' > $RESULT/invname

    test 1 == $(grep "inventory-" $RESULT/invname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
    fi
}

# assertPodExists checks that a pod with the passed podName and passed
# namespace exists in the cluster.
TIMEOUT_SECS=30
function assertPodExists {
    local podName=$1
    local namespace=$2

    kubectl wait --for=condition=Ready -n $namespace pod/$podName --timeout=${TIMEOUT_SECS}s > /dev/null 2>&1
    kubectl get po -n $namespace $podName -o name | awk '{print $1}' > $RESULT/podname

    test 1 == $(grep $podName $RESULT/podname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	ERROR+="ERROR: expected pod $namespace/$podName to exist\n"
    fi    
}

# assertPodNotExists checks that a pod with the passed podName and passed
# namespace DOES NOT exist in the cluster. Waits 20 seconds for pod
# termination if pod has not finished deleting.
function assertPodNotExists {
    local podName=$1
    local namespace=$2

    kubectl wait --for=delete -n $namespace pod/$podName --timeout=${TIMEOUT_SECS}s > /dev/null 2>&1
    kubectl get po -n $namespace $podName -o name > $RESULT/podname 2>&1
    
    test 1 == $(grep "(NotFound)" $RESULT/podname | wc -l);
    if [ $? == 0 ]; then
	echo -n '.'
    else
	echo -n 'E'
	ERROR+="ERROR: expected pod $namespace/$podName to not exist\n"
    fi    
}

# printResult prints the results of the previous assert statements
function printResult {
    if [ -z $ERROR ]; then
	echo "SUCCESS"
    else
	echo "ERROR"
    fi
    echo
    ERROR=""
}

# wait sleeps for the passed number of seconds.
function wait {
    local numSecs=$1

    sleep $numSecs
}

###########################################################################
#  Run tests
###########################################################################

unset RESOURCE_GROUP_INVENTORY

# Test 1: Basic ConfigMap init
# Creates ConfigMap inventory-template.yaml in "test-1" directory
echo "Testing basic ConfigMap init"
echo "kpt live init e2e/live/testdata/test-1"
${BASE}/kpt live init e2e/live/testdata/test-1 > $RESULT/status 2>&1
assertContains "namespace: test-namespace is used for inventory object"
assertContains "testdata/test-1/inventory-template.yaml"
printResult

# Copy the ConfigMap inventory template to the test-2 directory.
cp -f e2e/live/testdata/test-1/inventory-template.yaml e2e/live/testdata/test-2

# Test 2: Basic kpt live preview
# Preview run for "test-1" directory
echo "Testing initial preview"
echo "kpt live preview e2e/live/testdata/test-1"
${BASE}/kpt live preview e2e/live/testdata/test-1 > $RESULT/status
assertContains "namespace/test-namespace created (preview)"
assertContains "pod/pod-a created (preview)"
assertContains "pod/pod-b created (preview)"
assertContains "pod/pod-c created (preview)"
assertContains "4 resource(s) applied. 4 created, 0 unchanged, 0 configured"
assertContains "0 resource(s) pruned, 0 skipped"
printResult

# Test 3: Basic kpt live apply
# Apply run for "test-1" directory
echo "Testing basic apply"
echo "kpt live apply e2e/live/testdata/test-1"
${BASE}/kpt live apply e2e/live/testdata/test-1 > $RESULT/status
assertContains "namespace/test-namespace"
assertContains "pod/pod-a created"
assertContains "pod/pod-b created"
assertContains "pod/pod-c created"
assertContains "4 resource(s) applied. 3 created, 1 unchanged, 0 configured"
assertContains "0 resource(s) pruned, 0 skipped"
wait 2
# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertCMInventory "test-namespace" "4"
printResult

# Test 4: kpt live preview of apply/prune
# "test-2" directory is "test-1" directory with "pod-a" removed and "pod-d" added.
echo "Testing basic preview"
echo "kpt live preview e2e/live/testdata/test-2"
${BASE}/kpt live preview e2e/live/testdata/test-2 > $RESULT/status
assertContains "namespace/test-namespace configured (preview)"
assertContains "pod/pod-b configured (preview)"
assertContains "pod/pod-c configured (preview)"
assertContains "pod/pod-d created (preview)"
assertContains "4 resource(s) applied. 1 created, 0 unchanged, 3 configured (preview)"
assertContains "pod/pod-a pruned (preview)"
assertContains "1 resource(s) pruned, 0 skipped (preview)"
wait 2
# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertCMInventory "test-namespace" "4"
assertPodExists "pod-a" "test-namespace"
assertPodExists "pod-b" "test-namespace"
assertPodExists "pod-c" "test-namespace"
printResult

# Test 5: Basic kpt live apply/prune
# "test-2" directory is "test-1" directory with "pod-a" removed and "pod-d" added.
echo "Testing basic prune"
echo "kpt live apply e2e/live/testdata/test-2"
${BASE}/kpt live apply e2e/live/testdata/test-2 > $RESULT/status
assertContains "namespace/test-namespace unchanged"
assertContains "pod/pod-b unchanged"
assertContains "pod/pod-c unchanged"
assertContains "pod/pod-d created"
assertContains "4 resource(s) applied. 1 created, 3 unchanged, 0 configured"
assertContains "pod/pod-a pruned"
assertContains "1 resource(s) pruned, 0 skipped"
wait 2
# Validate resources in the cluster
# ConfigMap inventory with four inventory items.
assertCMInventory "test-namespace" "4"
assertPodExists "pod-b" "test-namespace"
assertPodExists "pod-c" "test-namespace"
assertPodExists "pod-d" "test-namespace"
assertPodNotExists "pod-a" "test-namespace"
printResult

# Test 6: Basic kpt live destroy
# "test-2" directory is "test-1" directory with "pod-a" removed and "pod-d" added.
echo "Testing basic destroy"
echo "kpt live destroy e2e/live/testdata/test-2"
${BASE}/kpt live destroy e2e/live/testdata/test-2 > $RESULT/status
assertContains "pod/pod-d deleted"
assertContains "pod/pod-c deleted"
assertContains "pod/pod-b deleted"
assertContains "namespace/test-namespace deleted"
assertContains "4 resource(s) deleted, 0 skipped"
# Validate resources NOT in the cluster
assertPodNotExists "pod-b" "test-namespace"
assertPodNotExists "pod-c" "test-namespace"
assertPodNotExists "pod-d" "test-namespace"
printResult

# Creates new inventory-template.yaml for "migrate-1" directory.
echo "kpt live init e2e/live/testdata/migrate-1"
rm -f e2e/live/testdata/migrate-1/inventory-template.yaml
${BASE}/kpt live init e2e/live/testdata/migrate-1 > $RESULT/status
assertContains "namespace: test-rg-namespace is used for inventory object"
assertContains "live/testdata/migrate-1/inventory-template.yaml"
printResult


###########################################################################
#  Tests with RESOURCE_GROUP_INVENTORY env var set
###########################################################################

export RESOURCE_GROUP_INVENTORY=1

# Test 7: kpt live apply ConfigMap inventory with RESOURCE_GROUP_INVENTORY set
# Applies resources in "migrate-1" directory.
echo "Testing kpt live apply with ConfigMap inventory"
echo "kpt live apply e2e/live/testdata/migrate-1"
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/migrate-1
${BASE}/kpt live apply e2e/live/testdata/migrate-1 > $RESULT/status
assertContains "namespace/test-rg-namespace unchanged"
assertContains "pod/pod-a created"
assertContains "pod/pod-b created"
assertContains "pod/pod-c created"
assertContains "4 resource(s) applied. 3 created, 1 unchanged, 0 configured"
assertContains "0 resource(s) pruned, 0 skipped"
# Validate resources in the cluster
assertPodExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
printResult

# Test 8: kpt live migrate from ConfigMap to ResourceGroup inventory
# Migrates resources in "migrate-1" directory.
echo "Testing migrate from ConfigMap to ResourceGroup inventory"
echo "kpt live migrate e2e/live/testdata/migrate-1"
${BASE}/kpt live migrate e2e/live/testdata/migrate-1 > $RESULT/status
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "updating Kptfile inventory values...success"
assertContains "retrieve the current ConfigMap inventory...success (4 inventory objects)"
assertContains "migrate inventory to ResourceGroup...success"
assertContains "deleting old ConfigMap inventory object...success"
assertContains "deleting inventory template file"
assertContains "inventory migration...success"
# Validate resources in the cluster
assertPodExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
printResult

# Test 9: kpt live preview with ResourceGroup inventory
# Previews resources in the "migrate-1" directory.
echo "Testing kpt live preview with ResourceGroup inventory"
echo "kpt live preview e2e/live/testdata/migrate-1"
${BASE}/kpt live preview e2e/live/testdata/migrate-1 > $RESULT/status
assertContains "namespace/test-rg-namespace configured (preview)"
assertContains "pod/pod-a configured (preview)"
assertContains "pod/pod-b configured (preview)"
assertContains "pod/pod-c configured (preview)"
assertContains "4 resource(s) applied. 0 created, 0 unchanged, 4 configured (preview)"
assertContains "0 resource(s) pruned, 0 skipped (preview)"
# Validate resources in the cluster
assertPodExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
printResult

# Test 10: kpt live apply/prune with ResourceGroup inventory
# "migrate-2" directory is the same as "migrate-1" with "pod-a" missing, and "pod-d" added.
echo "Testing kpt live apply/prune with ResourceGroup inventory"
echo "kpt live apply e2e/live/testdata/migrate-2"
cp -f e2e/live/testdata/migrate-1/Kptfile e2e/live/testdata/migrate-2
${BASE}/kpt live apply e2e/live/testdata/migrate-2 > $RESULT/status
assertContains "namespace/test-rg-namespace unchanged"
assertContains "pod/pod-a pruned"
assertContains "pod/pod-b unchanged"
assertContains "pod/pod-c unchanged"
assertContains "pod/pod-d created"
assertContains "4 resource(s) applied. 1 created, 3 unchanged, 0 configured"
assertContains "1 resource(s) pruned, 0 skipped"
# Validate resources in the cluster
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
assertPodExists "pod-d" "test-rg-namespace"
assertPodNotExists "pod-a" "test-rg-namespace"
printResult

# Test 11: kpt live destroy with ResourceGroup inventory
echo "Testing kpt destroy with ResourceGroup inventory"
echo "kpt live destroy e2e/live/testdata/migrate-2"
${BASE}/kpt live destroy e2e/live/testdata/migrate-2 > $RESULT/status
assertContains "pod/pod-d deleted"
assertContains "pod/pod-c deleted"
assertContains "pod/pod-b deleted"
assertContains "namespace/test-rg-namespace deleted"
assertContains "4 resource(s) deleted, 0 skipped"
assertPodNotExists "pod-b" "test-rg-namespace"
assertPodNotExists "pod-c" "test-rg-namespace"
assertPodNotExists "pod-d" "test-rg-namespace"
printResult

# Test 12: kpt live init for Kptfile (ResourceGroup inventory)
# initial Kptfile does NOT have inventory info
cp -f e2e/live/testdata/Kptfile e2e/live/testdata/migrate-3
echo "Testing kpt live init for Kptfile (ResourceGroup inventory)"
echo "kpt live init e2e/live/testdata/migrate-3"
${BASE}/kpt live init e2e/live/testdata/migrate-3 > $RESULT/status 2>&1
# Difference in Kptfile should have inventory data
diff e2e/live/testdata/Kptfile e2e/live/testdata/migrate-3/Kptfile > $RESULT/status 2>&1
assertContains "inventory:"
assertContains "namespace: test-rg-namespace"
assertContains "name: inventory-"
assertContains "inventoryID:"
printResult

# Test 13: kpt live migrate with missing inventory-template.yaml should fail
# "migrate-3" directory does not have an inventory-template.yaml
echo "Testing kpt live migrate with missing inventory-template.yaml should fail"
echo "kpt live migrate e2e/live/testdata/migrate-3"
rm -f e2e/live/testdata/migrate-3/inventory-template.yaml
${BASE}/kpt live migrate e2e/live/testdata/migrate-3 > $RESULT/status 2>&1
assertContains "inventory migration...failed"
printResult

# Test 14: kpt live migrate with no objects in cluster
# Add inventory-template.yaml to "migrate-3", but there are no objects in cluster.
cp -f e2e/live/testdata/inventory-template.yaml e2e/live/testdata/migrate-3
echo "Testing kpt live migrate with no objects in cluster"
echo "kpt live migrate e2e/live/testdata/migrate-3"
${BASE}/kpt live migrate e2e/live/testdata/migrate-3 > $RESULT/status 2>&1
assertContains "ensuring ResourceGroup CRD exists in cluster...success"
assertContains "updating Kptfile inventory values...values already exist...success"
assertContains "retrieve the current ConfigMap inventory...success (0 inventory objects)"
assertContains "deleting inventory template file:"
assertContains "e2e/live/testdata/migrate-3/inventory-template.yaml...success"
assertContains "inventory migration...success"
printResult

# Test 15: kpt live initial apply ResourceGroup inventory
echo "Testing kpt apply ResourceGroup inventory"
echo "kpt live apply e2e/live/testdata/migrate-3"
${BASE}/kpt live apply e2e/live/testdata/migrate-3 > $RESULT/status
assertContains "pod/pod-a created"
assertContains "pod/pod-b created"
assertContains "pod/pod-c created"
assertContains "0 resource(s) pruned, 0 skipped"
# Validate resources in the cluster
assertPodExists "pod-a" "test-rg-namespace"
assertPodExists "pod-b" "test-rg-namespace"
assertPodExists "pod-c" "test-rg-namespace"
printResult

echo

# Clean-up the k8s cluster
echo "Cleaning up cluster"
kind delete cluster
echo "FINISHED"

