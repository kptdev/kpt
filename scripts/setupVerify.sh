#!/bin/bash

function setupWorkspace() {
  TEST_HOME=$(mktemp -d)
  cd $TEST_HOME
}

function createOutputFile() {
  touch output.txt
}

function expectedOutput() {
  if [ "$(echo "$@")" != "$(cat output.txt)" ]
  then 
    echo "Expected:"
    echo "$(echo "$@")"
    echo "Received:"
    echo "$(cat output.txt)"
    exit 1
  else
    echo "Success"
  fi
}