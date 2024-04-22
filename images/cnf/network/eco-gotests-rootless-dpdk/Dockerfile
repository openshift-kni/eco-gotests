FROM quay.io/openshift-kni/dpdk:4.14.0

USER root
RUN dnf install -y net-tools iproute && dnf clean all && \
    cp /usr/local/bin/dpdk-testpmd /usr/bin/dpdk-testpmd && \
    setcap cap_net_admin=epi /usr/sbin/ip && \
    setcap cap_net_admin=epi /usr/sbin/route && \
    setcap cap_ipc_lock,cap_net_raw+ep /usr/bin/dpdk-testpmd

USER 1001
CMD ["/usr/libexec/s2i/usage"]
