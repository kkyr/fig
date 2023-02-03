#!/usr/bin/env bash

set -o pipefail

export CGO_ENABLED=1

if ! command -v golangci-lint &> /dev/null; then
  echo "Installing golangci-lint"
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.51.0
fi

echo -n "Running golangci-lint: "
ERRS=$(golangci-lint run "$@" 2>&1)
if [ $? -eq 1 ]; then
    echo "FAIL"
    echo "${ERRS}"
    echo
    exit 1
fi
echo "PASS"
echo