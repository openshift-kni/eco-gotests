# Build testcmd bin
FROM docker.io/library/golang:1.22 AS builder
RUN git clone https://github.com/kononovn/testcmd.git
WORKDIR ./testcmd
ENV CGO_ENABLED=0
RUN make build

FROM registry.access.redhat.com/ubi8/ubi:8.1

LABEL description="eco-gotests cnf network test client image"
RUN dnf update -y
RUN dnf install -y nginx iproute iputils procps-ng tcpdump
RUN dnf clean all
COPY --from=builder /go/testcmd/bin/testcmd /usr/bin/testcmd
