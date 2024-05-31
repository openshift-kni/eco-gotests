ARG BASE_IMG=quay.io/openshift-kni/dpdk
ARG BASE_TAG=4.14.0

FROM ${BASE_IMG}:${BASE_TAG}

USER root
RUN dnf install -y net-tools iproute && dnf clean all && \
    cp /usr/local/bin/dpdk-testpmd /usr/bin/dpdk-testpmd || true && \
    setcap cap_net_admin=epi /usr/sbin/ip && \
    setcap cap_net_admin=epi /usr/sbin/route && \
    setcap cap_ipc_lock,cap_net_raw+ep /usr/bin/dpdk-testpmd

USER 1001
CMD ["/usr/libexec/s2i/usage"]