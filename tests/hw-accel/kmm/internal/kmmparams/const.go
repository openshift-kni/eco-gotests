package kmmparams

const (
	// Label represents kmm that can be used for test cases selection.
	Label = "kmm"

	// KmmLogLevel custom loglevel of KMM related functions.
	KmmLogLevel = 90

	// MultistageContents represents the Dockerfile contents for multi stage build.
	MultistageContents = `ARG DTK_AUTO
FROM ${DTK_AUTO} as builder
ARG KERNEL_VERSION
ARG MY_MODULE
WORKDIR /build
RUN git clone https://github.com/cdvultur/kmm-kmod.git
WORKDIR /build/kmm-kmod
RUN cp kmm_ci_a.c {{.Module}}.c
RUN make

FROM registry.redhat.io/ubi8/ubi-minimal
ARG KERNEL_VERSION
ARG MY_MODULE
RUN microdnf -y install kmod

COPY --from=builder /etc/driver-toolkit-release.json /etc/
COPY --from=builder /build/kmm-kmod/*.ko /opt/lib/modules/${KERNEL_VERSION}/
RUN depmod -b /opt
`
	// SimpleKmodContents represents the Dockerfile contents for simple-kmod build.
	SimpleKmodContents = `FROM image-registry.openshift-image-registry.svc:5000/openshift/driver-toolkit
ARG KERNEL_VERSION
ARG KMODVER
WORKDIR /build/

RUN git clone https://github.com/cdvultur/simple-kmod.git && \
	cd simple-kmod && \
    make all       KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
    make install   KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
    mkdir -p /opt/lib/modules/$KERNEL_VERSION && \
    cp /lib/modules/$KERNEL_VERSION/simple-*.ko /lib/modules/$KERNEL_VERSION/modules.* /opt/lib/modules/$KERNEL_VERSION`

	// SecretContents template.
	SecretContents = `
{
  "auths": {
    "{{.Registry}}": {
      "auth": "{{.PullSecret}}",
      "email": ""
    }
  }
}
`
	// SimpleKmodFirmwareContents represents the Dockerfile contents for simple-kmod-firmware build.
	SimpleKmodFirmwareContents = `FROM image-registry.openshift-image-registry.svc:5000/openshift/driver-toolkit
ARG KVER
ARG KERNEL_VERSION
ARG KMODVER

WORKDIR /build/ 
RUN GIT_SSL_NO_VERIFY=1 git clone https://gitlab.cee.redhat.com/cvultur/simple-kmod.git && \
   cd simple-kmod && \
   make all       KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
   make install   KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
   mkdir -p /opt/lib/modules/$KERNEL_VERSION && \
   cp /lib/modules/$KERNEL_VERSION/simple-*.ko /lib/modules/$KERNEL_VERSION/modules.* /opt/lib/modules/$KERNEL_VERSION

RUN mkdir /firmware
RUN echo -n "simple_kmod_firmware validation string" >> /firmware/simple_kmod_firmware.bin
`
)
