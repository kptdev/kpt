#!/bin/bash

function setupWorkspace() {
  TEST_HOME=$(mktemp -d)
  cd $TEST_HOME
}

function createOutputFile(){
  touch output.txt
}

function expectedOutput() {
  if [ "$(echo "$@")" == "$(cat output.txt)" ]; then echo 0; else echo 1; fi
}