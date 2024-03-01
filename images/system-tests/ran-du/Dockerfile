ARG BASE_IMG=quay.io/ocp-edge-qe/eco-gotests
ARG BASE_TAG=latest

FROM ${BASE_IMG}:${BASE_TAG}

# Install helm
ADD https://get.helm.sh/helm-v3.13.2-linux-amd64.tar.gz /tmp
RUN tar -C /tmp/ -zxvf /tmp/helm-v3.13.2-linux-amd64.tar.gz && \
    mv /tmp/linux-amd64/helm /usr/local/bin && \
    rm /tmp/helm-v3.13.2-linux-amd64.tar.gz

# Install OC binary
ADD https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz /tmp
RUN tar -C /usr/local/bin -xzf /tmp/openshift-client-linux.tar.gz && rm /tmp/openshift-client-linux.tar.gz

# Install DU dependencies
RUN dnf install -y ansible-core make python3.11-pip

RUN pip3 install -r images/system-tests/ran-du/requirements.txt &&\
    ansible-galaxy install -r images/system-tests/ran-du/requirements.yml
