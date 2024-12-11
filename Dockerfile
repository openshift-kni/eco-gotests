FROM registry.access.redhat.com/ubi9/ubi:latest

ARG GO_VER=go1.23.2
ARG GINKGO_VER=ginkgo@v2.20.2
ARG CONTAINERUSER=testuser

LABEL description="eco-gotests development image"
LABEL go.version=${GO_VER}
LABEL ginkgo.version=${GINKGO_VER}
LABEL container.user=${CONTAINERUSER}

ENV GOPATH="/usr/local/go"
ENV PATH "$PATH:$GOPATH/bin"

RUN dnf install -y tar gcc make && \
    dnf clean metadata packages && \
    arch=$(arch | sed s/aarch64/arm64/ | sed s/x86_64/amd64/) && \
    curl -Ls https://go.dev/dl/${GO_VER}.linux-${arch}.tar.gz |tar -C /usr/local -xzf -  && \
    go install github.com/onsi/ginkgo/v2/${GINKGO_VER} && \
    useradd -U -u 1000 -m -d /home/${CONTAINERUSER} -s /usr/bin/bash ${CONTAINERUSER} && \
    mkdir /home/${CONTAINERUSER}/eco-gotests && \
    echo 'RUN done'

USER ${CONTAINERUSER}

WORKDIR /home/${CONTAINERUSER}/eco-gotests
COPY --chown=${CONTAINERUSER}:${CONTAINERUSER} . .

ENTRYPOINT ["scripts/test-runner.sh"]
