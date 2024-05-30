# Build testcmd bin
FROM docker.io/library/golang:1.22 AS builder
RUN git clone https://github.com/kononovn/testcmd.git
WORKDIR ./testcmd
ENV CGO_ENABLED=0
RUN make build

FROM registry.access.redhat.com/ubi8/ubi:8.9

LABEL description="eco-gotests cnf network test client image"
RUN dnf install -y nginx iproute iputils procps-ng ethtool shadow-utils libpcap-14:1.9.1-5.el8.x86_64
RUN curl http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/tcpdump-4.9.3-1.el8.x86_64.rpm -o tcpdump-4.9.3-1.el8.x86_64.rpm
RUN rpm -i tcpdump-4.9.3-1.el8.x86_64.rpm
RUN dnf clean all
COPY --from=builder /go/testcmd/bin/testcmd /usr/bin/testcmd
