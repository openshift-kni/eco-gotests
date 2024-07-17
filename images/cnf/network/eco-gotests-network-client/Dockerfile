FROM registry.access.redhat.com/ubi9/ubi:latest

LABEL description="eco-gotests cnf network test client image"
RUN dnf update -y
RUN dnf install -y nginx iproute iputils procps-ng shadow-utils libpcap
RUN curl -s https://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/tcpdump-4.99.0-9.el9.x86_64.rpm -o tcpdump-4.99.0-9.el9.x86_64.rpm
RUN rpm -i tcpdump-4.99.0-9.el9.x86_64.rpm
RUN dnf clean all
