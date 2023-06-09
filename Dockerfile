FROM registry.access.redhat.com/ubi8/ubi:latest

ARG GO_VER=go1.19.10.linux-amd64.tar.gz
ARG GINKGO_VER=ginkgo@v2.10.0

LABEL description="eco-gotests development image"
LABEL go.version=${GO_VER}
LABEL ginkgo.version=${GINKGO_VER}

ENV PATH "$PATH:/usr/local/go/bin:/root/go/bin"
RUN dnf install -y tar gcc make && \
    dnf clean metadata packages && \
    curl -Ls https://go.dev/dl/${GO_VER} |tar -C /usr/local -xzf -  && \
    go install github.com/onsi/ginkgo/v2/${GINKGO_VER}
# RUN go get github.com/onsi/gomega/...

WORKDIR /workspace
COPY . .

ENTRYPOINT ["scripts/test-runner.sh"]
