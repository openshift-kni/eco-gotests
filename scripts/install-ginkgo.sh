#!/usr/bin/env bash

REQUIRED_GINKGO_VERSION="$(grep '/ginkgo' go.mod | cut -d ' ' -f2 | cut -c 2-)"
INSTALLED_GINKGO_VERSION="$(ginkgo version | { read -r _ _ v _; echo "${v}"; })"

GOPATH="${GOPATH:-/root/go}"
export PATH=$PATH:$GOPATH/bin

function version { echo "$@" | awk -F. '{ printf("%d%03d%03d%03d\n", $1,$2,$3,$4); }'; }

if [ "$(version "${INSTALLED_GINKGO_VERSION}")" -ge "$(version "${REQUIRED_GINKGO_VERSION}")" ]; then
    echo "Version is up to date"
else
    echo "Downloading ginkgo tool"
	go install "$(awk '/ginkgo/ {printf "%s/ginkgo@%s", $1, $2}' go.mod)"
fi
