#!/usr/bin/env bash

GOPATH="${GOPATH:-~/go}"
export PATH=$PATH:$GOPATH/bin

if ! command -v ginkgo ; then
  echo "Downloading ginkgo tool"
  go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4
fi
