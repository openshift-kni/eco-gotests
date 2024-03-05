FROM registry.access.redhat.com/ubi8/ubi:latest

ARG GO_VER=go1.20.11
ARG GINKGO_VER=ginkgo@v2.13.2

LABEL description="eco-gotests development image"
LABEL go.version=${GO_VER}
LABEL ginkgo.version=${GINKGO_VER}

ENV PATH "$PATH:/usr/local/go/bin:/root/go/bin"
RUN dnf install -y tar gcc make && \
    dnf clean metadata packages && \
    arch=$(arch | sed s/aarch64/arm64/ \ 
                | sed s/x86_64/amd64/) && \
    curl -Ls https://go.dev/dl/${GO_VER}.linux-${arch}.tar.gz |tar -C /usr/local -xzf -  && \
    go install github.com/onsi/ginkgo/v2/${GINKGO_VER}

WORKDIR /workspace
COPY . .

ENTRYPOINT ["scripts/test-runner.sh"]
