FROM registry.access.redhat.com/ubi8/ubi:8.1

LABEL description="eco-gotests cnf network test client image"
RUN dnf update -y
RUN dnf install -y nginx iproute iputils
RUN dnf clean all
