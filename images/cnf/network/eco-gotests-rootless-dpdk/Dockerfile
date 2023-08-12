FROM registry-proxy.engineering.redhat.com/rh-osbs/dpdk-base:v4.14.0-6

USER root
RUN dnf install -y net-tools iproute && dnf clean all
RUN setcap cap_net_admin=epi /usr/sbin/ip
RUN setcap cap_net_admin=epi /usr/sbin/route
RUN setcap cap_net_raw,cap_sys_resource,cap_ipc_lock+ep /usr/bin/dpdk-testpmd
USER 1001
CMD ["/usr/libexec/s2i/usage"]
