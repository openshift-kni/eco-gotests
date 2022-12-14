FROM registry.access.redhat.com/ubi8/ubi:8.1

ARG GO_VER=go1.18.8.linux-amd64.tar.gz
ARG GINKGO_VER=ginkgo@v2.1.4

RUN dnf install -y wget tar && \
    wget https://go.dev/dl/${GO_VER} && \
    tar -xf ${GO_VER} -C /usr/local && \
    rm -f ${GO_VER}

ENV PATH "$PATH:/usr/local/go/bin"

RUN go install github.com/onsi/ginkgo/v2/${GINKGO_VER}
# RUN go get github.com/onsi/gomega/...

LABEL description="eco-gotests development image"
LABEL go.version=${GO_VER}
LABEL ginkgo.version=${GINKGO_VER}

WORKDIR /workspace

ENTRYPOINT ["/bin/bash", "-c"]
CMD ["go version"]
