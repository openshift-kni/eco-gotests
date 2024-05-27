# Build testcmd bin
FROM docker.io/library/golang:1.22 AS builder
RUN git clone https://github.com/kononovn/testcmd.git
WORKDIR ./testcmd
ENV CGO_ENABLED=0
RUN make build

FROM registry.access.redhat.com/ubi8/ubi:8.9

LABEL description="eco-gotests cnf network test client image"
RUN dnf config-manager --add-repo="http://download.engineering.redhat.com/released/rhel-6-7-8/rhel-8/RHEL-8/8.9.0/AppStream/x86_64/os"
RUN dnf update -y
RUN dnf install -y nginx iproute iputils procps-ng tcpdump ethtool
RUN dnf clean all
COPY --from=builder /go/testcmd/bin/testcmd /usr/bin/testcmd
