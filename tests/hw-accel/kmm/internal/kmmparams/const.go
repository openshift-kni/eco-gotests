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
)
