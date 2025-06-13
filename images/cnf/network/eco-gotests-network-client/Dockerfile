# Build testcmd bin
FROM docker.io/library/golang:1.23 AS builder
RUN git clone https://github.com/kononovn/testcmd.git
WORKDIR ./testcmd
ENV CGO_ENABLED=0
RUN make build

FROM registry.access.redhat.com/ubi9/ubi:latest

LABEL description="eco-gotests cnf network test client image"
RUN dnf install -y nginx iproute iputils procps-ng ethtool shadow-utils libpcap net-tools nmap
RUN curl -s https://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/tcpdump-4.99.0-9.el9.x86_64.rpm -o tcpdump-4.99.0-9.el9.x86_64.rpm
RUN rpm -i tcpdump-4.99.0-9.el9.x86_64.rpm
RUN dnf clean all
RUN echo 'server {location /clientip {default_type text/plain;return 200 "Client IP Address: $remote_addr\n";} location /hostname {default_type text/plain;return 200 "You reached host: $hostname\n";} location /serverip {default_type text/plain;return 200 "Server IP Address: $server_addr\n";} location /checknet {default_type text/plain;return 200 "Client IP: $remote_addr\nServer Hostname: $hostname\nServer IP: $server_addr\n";}}' > /etc/nginx/conf.d/custom.conf
COPY --from=builder /go/testcmd/bin/testcmd /usr/bin/testcmd