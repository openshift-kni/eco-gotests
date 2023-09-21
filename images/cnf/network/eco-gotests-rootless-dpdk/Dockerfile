FROM registry-proxy.engineering.redhat.com/rh-osbs/dpdk-base:v4.14.0-6

USER root
RUN dnf install -y net-tools iproute && dnf clean all
RUN setcap cap_net_admin=epi /usr/sbin/ip
RUN setcap cap_net_admin=epi /usr/sbin/route
RUN setcap cap_ipc_lock,cap_net_raw+ep /usr/bin/dpdk-testpmd

# Add supplementary group 801 to user 1001 in order to use the VFIO device in a non-privileged pod.
RUN usermod -aG hugetlbfs default

USER 1001
CMD ["/usr/libexec/s2i/usage"]
