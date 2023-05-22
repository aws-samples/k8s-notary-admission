#!/usr/bin/env bash

trap 'catch $? $LINENO' ERR
catch() {
  echo "Error $1 occurred on $2"
}

KUBECTL="kubectl"

TEST_DIR=${1:-"test"}

pushd "${TEST_DIR}"

${KUBECTL} apply -f .

# sleep 5

# ${KUBECTL} delete -f .

popd
