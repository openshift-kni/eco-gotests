FROM registry.access.redhat.com/ubi9/ubi

RUN dnf -y install iperf3 iputils && \
    dnf clean all

CMD [ "/bin/bash" ]
