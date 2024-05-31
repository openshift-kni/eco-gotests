
BASE_IMG ?= eco-gotests
BASE_TAG ?= latest

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on
GO_PACKAGES=$(shell go list ./... | grep -v vendor)
.PHONY: lint \
        deps-update \
        vet
vet:
	go vet ${GO_PACKAGES}

lint:
	@echo "Running go lint"
	scripts/golangci-lint.sh

deps-update:
	go mod tidy && \
	go mod vendor

install-ginkgo:
	scripts/install-ginkgo.sh

build-docker-image:
	@echo "Building docker image"
	podman build -t "${BASE_IMG}:${BASE_TAG}" -f Dockerfile

build-docker-image-ran-du: build-docker-image
	@echo "Building docker image for RAN DU tests"
	podman build --build-arg=BASE_IMG="${BASE_IMG}" --build-arg=BASE_TAG="${BASE_TAG}" -t eco-gotests-ran-du:latest -f images/system-tests/ran-du/Dockerfile

install: deps-update install-ginkgo
	@echo "Installing needed dependencies"

run-tests:
	@echo "Executing eco-gotests test-runner script"
	scripts/test-runner.sh
