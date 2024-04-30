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
RUN make KVER=${KERNEL_VERSION}

FROM registry.redhat.io/ubi9/ubi-minimal
ARG KERNEL_VERSION
ARG MY_MODULE
RUN microdnf -y install kmod

COPY --from=builder /etc/driver-toolkit-release.json /etc/
COPY --from=builder /build/kmm-kmod/*.ko /opt/lib/modules/${KERNEL_VERSION}/
RUN depmod -b /opt ${KERNEL_VERSION}
`
	// SimpleKmodContents represents the Dockerfile contents for simple-kmod build.
	SimpleKmodContents = `ARG DTK_AUTO
FROM ${DTK_AUTO} as builder
ARG KERNEL_VERSION
ARG KMODVER
WORKDIR /build/

RUN git clone https://github.com/cdvultur/simple-kmod.git && \
	cd simple-kmod && \
    make all       KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
    make install   KVER=$KERNEL_VERSION KMODVER=$KMODVER

FROM registry.redhat.io/ubi9/ubi-minimal
ARG KERNEL_VERSION
ARG MY_MODULE
RUN microdnf -y install kmod

COPY --from=builder /etc/driver-toolkit-release.json /etc/
COPY --from=builder /lib/modules/$KERNEL_VERSION/simple-*.ko /opt/lib/modules/${KERNEL_VERSION}/
COPY --from=builder /lib/modules/$KERNEL_VERSION/modules.* /opt/lib/modules/${KERNEL_VERSION}/
RUN depmod -b /opt ${KERNEL_VERSION}
`

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
	SimpleKmodFirmwareContents = `ARG DTK_AUTO
FROM ${DTK_AUTO} as builder
ARG KVER
ARG KERNEL_VERSION
ARG KMODVER

WORKDIR /build/ 
RUN git clone https://github.com/cdvultur/simple-kmod.git && \
   cd simple-kmod && \
   make all       KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
   make install   KVER=$KERNEL_VERSION KMODVER=$KMODVER && \
   mkdir -p /opt/lib/modules/$KERNEL_VERSION && \
   cp /lib/modules/$KERNEL_VERSION/simple-*.ko /lib/modules/$KERNEL_VERSION/modules.* /opt/lib/modules/$KERNEL_VERSION

RUN mkdir /firmware
RUN echo -n "simple_kmod_firmware validation string" >> /firmware/simple_kmod_firmware.bin
`
	// LocalMultiStageContents represents the Dockerfile contents for multi stage build using local registry.
	LocalMultiStageContents = `FROM image-registry.openshift-image-registry.svc:5000/openshift/driver-toolkit as builder
ARG KERNEL_VERSION
ARG MY_MODULE
WORKDIR /build
RUN git clone https://github.com/cdvultur/kmm-kmod.git
WORKDIR /build/kmm-kmod
RUN cp kmm_ci_a.c {{.Module}}.c
RUN make

FROM registry.redhat.io/ubi9/ubi-minimal
ARG KERNEL_VERSION
ARG MY_MODULE
RUN microdnf -y install kmod

COPY --from=builder /etc/driver-toolkit-release.json /etc/
COPY --from=builder /build/kmm-kmod/*.ko /opt/lib/modules/${KERNEL_VERSION}/
RUN depmod -b /opt
`

	//nolint:lll
	// SigningCertBase64 represents cert used for module signing.
	SigningCertBase64 = `MIIFkTCCA3mgAwIBAgIUQM6ZTI8oUcyIYOwweq0rcs+f/8YwDQYJKoZIhvcNAQELBQAwUjEQMA4GA1UECgwHY2R2dGVzdDEcMBoGA1UEAwwTY2R2dGVzdCBzaWduaW5nIGtleTEgMB4GCSqGSIb3DQEJARYRY2hyaXNwQHJlZGhhdC5jb20wIBcNMjIxMTI1MTM1NzMzWhgPMjEyMjExMDExMzU3MzNaMFIxEDAOBgNVBAoMB2NkdnRlc3QxHDAaBgNVBAMME2NkdnRlc3Qgc2lnbmluZyBrZXkxIDAeBgkqhkiG9w0BCQEWEWNocmlzcEByZWRoYXQuY29tMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuZ2SAeiQkixbP+rb648eKKnwA+sO2f4w7hPouy/wtpjaq63wDYUXEM5NIqDEkx4DAzlqXrWhvClQ7dzF04zeRFFWAbafcBupFPofKUhF+Qx/4bMAN7DRFwQQHWHfidwnzDM31mLEIN4kGdKp1gf6glBbBsvZwDPm7rGRo+ZyxaUnuzCgyl3+lYxiJ9Zl5h223/smgniZKX9oI/Jxf7af0N0E9Jtjbs8f+N15oO24uEd4/NHm4of6VRqaxDiTR6u+FP+pldEhAvTPd5qAjrI+9nm8ha+Myob2x3ufbpfPpKp4lwbBJLBzAmsJwi/6183mrP4kBwnrXbFPBwATg5JFWzLlbYC9pp88Q3Kzff8ml88jo4Qzj0F8WWNii7eVHZkefuoMhhu7RNUnx/C0hNuGsj9HXK2r+ufySr6BMgX5EF3HgwBZNyxYVWVToN9mYAluDRWAy6iJaW+IWFkw8ZdP1d5Emh+qlCJOGYdeFl7rVRTR5yc7pG2aTxaRGhsi5eWKtDge1Nwalg2MPMdprIamfOn+eDJEWdErA4MIlkz2NgCm5d7nnTP9GWiCJ5RTF+VXjhvsOkSSiyQaY05vCoN9vwOAyLIV1geJfDqooJuLPY1qQjjZu5vALt/t4855MPa2dxWULkvrpNaor3vSxk12yu4Ir88DG1Ahc5sgCmdk5FcCAwEAAaNdMFswDAYDVR0TAQH/BAIwADALBgNVHQ8EBAMCB4AwHQYDVR0OBBYEFJ+Uyk+hqFQCf7AIP8nOg0O77iwaMB8GA1UdIwQYMBaAFJ+Uyk+hqFQCf7AIP8nOg0O77iwaMA0GCSqGSIb3DQEBCwUAA4ICAQCRX+ywa0tpMWLqidtKvTEg38kk5qGe6c66ySEX8jYiHOLk1IsmlxY9tWoUxqmPUTHzBL6a+LLWo529JuZNj1cIjm6RxTa+N82W0E91IjXtU62bNutN5bf+LcJL1YyK50/KtYxYUXVnITQ+9AC15snTQgppQwj5nlc4F72bSoNB8++K1rvI6jmFrZo9xJg8z3sRu5v/3UCfcogRAuF6HXeUlD1dcY3sB6rf19w8xoPkPz0iTG+qUy1lZbjLyn2+cq86ZUgPjvdvJB5l3f0b6a/yMWuCZAg0fHTC6ak/v5sq/HOPQGoF+MCAPt8XsgGIR+AErzcQQpx+agnUXdGKY1FZMJS3xlEPDe9Ud93HpWv33ZFv/dGZYHqK+UeO2CfBRj4md75euS8eEhvp5FZZQTMm7XBYrNqS2LSCuRiIVyDG/UtIv6qwy9FaxaFVNX9slS44XPeOMZ5JNki5A7kSHOldVVbTHFoQ/yFkLQZLEYhuqh9tD19wR1McwBcOx0rWZKC/6nYQq1swvoR10I5fQaOtcf0YMvRrglLEAfQTWGC/G2wf8VIkCk9A+xvkk1JUemlimNKdCFbIiWp675pBje99DuO+zsCuGPla0ORoOIWRwDBRxBeFN2ObAIpAEojG3P9wOYuUz1byxkfVXZMRYaxFCq7HAmwbznyRtTsYskT7jw==`

	//nolint:lll
	// SigningKeyBase64 represents key used for module signing.
	SigningKeyBase64 = `LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUpRd0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQ1Mwd2dna3BBZ0VBQW9JQ0FRQzVuWklCNkpDU0xGcy8KNnR2cmp4NG9xZkFENnc3Wi9qRHVFK2k3TC9DMm1OcXJyZkFOaFJjUXprMGlvTVNUSGdNRE9XcGV0YUc4S1ZEdAozTVhUak41RVVWWUJ0cDl3RzZrVStoOHBTRVg1REgvaHN3QTNzTkVYQkJBZFlkK0ozQ2ZNTXpmV1lzUWczaVFaCjBxbldCL3FDVUZzR3k5bkFNK2J1c1pHajVuTEZwU2U3TUtES1hmNlZqR0luMW1YbUhiYmYreWFDZUprcGYyZ2oKOG5GL3RwL1EzUVQwbTJOdXp4LzQzWG1nN2JpNFIzajgwZWJpaC9wVkdwckVPSk5IcTc0VS82bVYwU0VDOU05Mwptb0NPc2o3MmVieUZyNHpLaHZiSGU1OXVsOCtrcW5pWEJzRWtzSE1DYXduQ0wvclh6ZWFzL2lRSENldGRzVThICkFCT0Rra1ZiTXVWdGdMMm1uenhEY3JOOS95YVh6eU9qaERPUFFYeFpZMktMdDVVZG1SNSs2Z3lHRzd0RTFTZkgKOExTRTI0YXlQMGRjcmF2NjUvSkt2b0V5QmZrUVhjZURBRmszTEZoVlpWT2czMlpnQ1c0TkZZRExxSWxwYjRoWQpXVER4bDAvVjNrU2FINnFVSWs0WmgxNFdYdXRWRk5Ibkp6dWtiWnBQRnBFYUd5TGw1WXEwT0I3VTNCcVdEWXc4CngybXNocVo4NmY1NE1rUlowU3NEZ3dpV1RQWTJBS2JsM3VlZE0vMFphSUlubEZNWDVWZU9HK3c2UkpLTEpCcGoKVG04S2czMi9BNERJc2hYV0I0bDhPcWlnbTRzOWpXcENPTm03bThBdTMrM2p6bmt3OXJaM0ZaUXVTK3VrMXFpdgplOUxHVFhiSzdnaXZ6d01iVUNGem15QUtaMlRrVndJREFRQUJBb0lDQUdlR0FwRWQ3TVFhQ1NxMUNzTnVweHQyCjdHN0RBeDJDTURnWTBvelVTQ1VQUzZiRTlFWVdISzg2NExxWGdBdXRpWENhN1VDMWZPYzlBKzRtWEErSldjNHcKcnc5MWs2Y3JSanAwNWp4WUd1NjBrZjZIbjI5d0pxMzNqaVZWR1NBQ3NnK3A0VktvODBxQm14RHI5ZDF4eGI0NApvd3cvVmN0bEp6K3B1ZGo0QnJ0RlNjdVZhNWh5cERNMGdPcEhJYjRlaElpWHd2cXgySHJVbkhpd0Y2MXJaZWRQCjNodXZMdDMybDVadHNCQ1poWlVDQi9DNkFWUUhFdyswTHUyUndGTVE3b3MzckpYcWRjYjRVMTRaeGVuUWNGMTMKdURXM200YTVqNW5pMFNFVmVySVVnZjE1dnU5L21pV1hIWEk0NEdiTWFBUEVHc1B5UEZ1ZE1MNXV4SE4xZVVoWQpCaXhvc3c0dDJUamtXK1hWMDBPcWlGV2JlanM2N0xPN0JQUWRCYTZTNG5yNyt2Qksxc1B4NkdMRGNPUWl4QjNrCldhM0ZpUkFOazBCNUdEU2t0RDl5SXdUTFVYZ2tIWmloZFF2VUhWMmNicGplM1Y1cVhnVUZQcGk0dHNTaGVXbVIKRzlOTE85ZlBxVnBoVGdqTEFUV0J4dG1TUU1pYmVDZDc5MVkySVc4bXRGNENuU29ZYVlFOVhsdEZxOVpBQ1lIdwpNdk9tYm5OeFlUN2lKdmgvVmUzclh0dzNLMURGOXZhT1RFSEV5dWhlaUp0K0xhQ1ZncGh6TjkxL0cxU21ieUlWCmgybUlabWxMWHlDTVpva2hmNUNIZGFtbUhsSTRERStmdS90ZUxYdVhLKzQ3T2xMZ3pNVjhSS213MmtMREQ5a3oKWmg2c1F0VFdKaWlib3pMbktVb0JBb0lCQVFEdTJBYjlab09lVHFUVU1QUExhcHZQdzFBS1ZQTjcvcEswNnF5cgo2V2RJUWNONjJlTnpQTjVWYkxMNE8rQm85UXAxMXEvN3Y0RXNMYnNLL25EMExJU05WRi8ybGJzSS9ORDIrVGZjClZMRGhaQUtodXh5eXZHc0tGL0dDWVJVZXF3K0NORHIyaDNXcDlSelRDR25qNERtMUcwNmFYTGdkVXNZbXAwdGgKdXAxUHRCb25ubWlvMUFLdWhzSHE1cUQ0Q09tUFJzK3J2ZCs3MWlmZXhmSzJBMEZTazRZNWhUQTFZb3RsK0d4WAprRk45OGdSby9wQWNzN0w3eG84WkdseDY2Wm1ZaC9Gd3NGUUVnMGc0NHVHMldCMUtlUjRJZW0xdjRQTVE3WGxpCkkxVFdmZk1FMXhWSzUrM1VkenZteDdiYnJMWlNocmhpT29EOEZwNEZrWWVFclFyaEFvSUJBUURHOHNGWVhvcGYKU1ZsaUlkK0YrbkoxdlRqeWdMWXViMzk0T2dVK2VpZ2J2UmNjSHNWYkxlVGx1d3lZYVNBR0NaeE1LUndXY3dVVwp5d1MzaWtjcm4xdHFkVlJiSVlqZnNIc3hXbCtJNm5PSE5FNWd5bHBiYUY1YU5URkhqQ1BMb3k5SkpPZGdLaG4yCmt4Tmd2RC9qUHFhV1FYaGhwOFVRSHJsOGNiaHhMLzRlNnlHNE80ZHluc0ZFZDFQeTBXZEFMUEVpNEtVbmJaWW4KeWRpbDlNdU0vSnkwUXZJZkkvUDJGc3dOMWhsNDJSWkExMkJOSzZSck1pekhIUlUxeGZJWTZiTXlVTUlUMkQ2MQpJOURBNVhuUGRJMzVmK1crek5kWCtreGRkY2Erdjh5OXVVcGNVYVZXTHFWTHZzWFo3cUMvbUxBOFF3ZlRNZ3NOCnBlQ2NCNHU0eDA0M0FvSUJBUURWbnZIaGd1Y0ZtR0ZrUjhSRms3eDRQc0EvL1dzbzQ3QmpqK0dRZ05tWGp2by8KenRIWUtBRFRkcjA3dUpJbVRjUmxVUGRsdXdyVmNMRnlTOURMRTJZYTRmUlNuK2tCU04yOWgzbW0zemkwM3JaYgo0UGJ5QmdQV3EwT2UwU1lLb0FUbTk4QWs3MU1XQjkwWUF0WnlzZ0hyTWRsRHh0b0ZvQnNLUjJic3FmUTViV1JYCk94OXdvTzhsR2ZJbkhzK3FDSTZkVDBBKzR6eFF6R0lzcGU5SFMvSUk0Vm1UNk1RTmUyNGliZWE1Q3FVaWFHdjUKWEhXWXRrREhYL2h0QTE3anNEdG9hVzVRMCtUUmhIbjhKekNwM25XVVBtL1dOV25jUHQ0bnJiNTdRQTZKS1cxSwpUdlVFWWh3ZGcrZFhxaGlxc2ZjQUtPNlJMTEpneGZuZ0VTR2NVUWtCQW9JQkFFdWxUQnprdmFwamdtZ014eWZ6ClJZZzlMYVVQaWJYNFVUaU9ueVhWWHVERk1qOVA5K3ltYzYxaVJQVENyQmwvbC8xaGVEdVUrbTlqUEdUcFlBeFgKS0hRL0xwY0VGajR2cFhmcmkvM01YNmNlSFZzeU5jOGh6Ulp4dVU0aUhBNDIreWpOcm1oak9jSUd4RXg0NTdYcApRWUJLWHBLTEx5UGsrdFExalZNRVU1VEFCTzgvTzA0NnpQUFNoNG9CVTBnVWpvK2JhVkNubTN0L2hTLzg5MVNoClRKaENDRHdNK0pzdXFlSHM4WHlBMXJSSzhHUUhYeG9mVnVWU3lwaktyallJemtrb2FkTVAyekFXOFM0WFV3eXQKbmJvcmhsalpIRnhvWUpiOHpGZ0ZKNzFQOGRWT2VoWmQ0QjMvNk16bnJobUwzaDdid2VMczJVVVVPR1k3ZkVZRApDbHNDZ2dFQkFJMklwSzlic09GOUVhNTBNcEFCdlBxRDZ1bG05c0xlK1RHaStSR2NDLzVoZEZDNG1BbTFiWDE3ClJMU3g4ZktCaGtkdjN6UU1zdExpd1d0V2xybFA5dkVhbWlrYndUT1hCM1hrQW9mK0ZHSzZPNm5vbHZ1MnJzdFUKak9hZEs0aFRjdWNJM0NBRWkzenZTZWR2Q1k0ZHlHQVR5VGcyTGNCclgvcDl0T2YrQzNHc05jazFxSThzTVhZbworUTVSa3pnNGlBK1BpMWUwOG5wcmxZUlF2a3hBaHQzOUZyZ283TWZ1clV3eHMxN3VGZzJFUE9xRTFDZndyTkRiCnRxc2M4c1VYNkZaN2dFek1kak8vd2kvYmJpbVR3NERldUVxTEdKSWJmSGZiZGk5U29lbEtrYVptUVY0RTY1VXUKd0t1N0l6L0h3ZWlLVVJqc0dxekJxcjFraTFaUW9wST0KLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo=`

	// KmmScannerDockerfile represents dockerfile used to run clamav on KMM images.
	KmmScannerDockerfile = `ARG OPERATOR_IMAGE
ARG MUST_GATHER
ARG SIGN
ARG WORKER
ARG RBAC_IMAGE

FROM ${OPERATOR_IMAGE} as operator
FROM ${MUST_GATHER} as must-gather
FROM ${SIGN} as sign
FROM ${WORKER} as worker
FROM ${RBAC_IMAGE} as rbac

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8-1072
RUN rpm -ivh https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm && \
    microdnf -y --setopt=tsflags=nodocs install \
    clamav \
    clamd \
    clamav-update && \
    microdnf clean all

WORKDIR /operator
COPY --from=operator . .

WORKDIR /must-gather
COPY --from=must-gather . .

WORKDIR /sign
COPY --from=sign . .

WORKDIR /worker
COPY --from=worker . .

WORKDIR /rbac
COPY --from=rbac . .

WORKDIR /
RUN freshclam
RUN clamscan -v -a --bell --log=/operator.log -r -z /operator
RUN clamscan -v -a --bell --log=/must-gather.log -r -z /must-gather
RUN clamscan -v -a --bell --log=/sign.log -r -z /sign
RUN clamscan -v -a --bell --log=/worker.log -r -z /worker
RUN clamscan -v -a --bell --log=/rbac.log -r -z /rbac
RUN chmod o+r *.log

`
)

const (
	// LabelSuite represents kmm label that can be used for test cases selection.
	LabelSuite = "module"
	// LabelSanity represents kmm label for short-running tests used for test case selection.
	LabelSanity = "kmm-sanity"
	// LabelLongRun represent kmm label for long-running tests used for test case selection.
	LabelLongRun = "kmm-longrun"
	// KmmOperatorNamespace represents the namespace where KMM is installed.
	KmmOperatorNamespace = "openshift-kmm"
	// KmmHubOperatorNamespace represents namespace of the operator.
	KmmHubOperatorNamespace = "openshift-kmm-hub"
	// DeploymentName represents the name of the KMM operator deployment.
	DeploymentName = "kmm-operator-controller"
	// WebhookDeploymentName represents the name of the Webhook server deployment.
	WebhookDeploymentName = "kmm-operator-webhook-server"
	// HubDeploymentName represents the name of the KMM HUB deployment.
	HubDeploymentName = "kmm-operator-hub-controller"
	// HubWebhookDeploymentName represents the name of the HUB Webhook server deployment.
	HubWebhookDeploymentName = "kmm-operator-hub-webhook-server"
	// BuildArgName represents kmod key passed to kmm-ci example.
	BuildArgName = "MY_MODULE"
	// RelImgMustGather represents identifier for must-gather image in operator environment variables.
	RelImgMustGather = "MUST_GATHER"
	// RelImgSign represents identifier for sign image in operator environment variables.
	RelImgSign = "SIGN"
	// RelImgWorker represents identifier for worker image in operator environment variables.
	RelImgWorker = "WORKER"
	// ModuleNodeLabelTemplate represents template of the label set on a node for a Module.
	ModuleNodeLabelTemplate = "kmm.node.kubernetes.io/%s.%s.ready"
	// DevicePluginNodeLabelTemplate represents template label set by KMM on a node for a Device Plugin.
	DevicePluginNodeLabelTemplate = "kmm.node.kubernetes.io/%s.%s.device-plugin-ready"
	// UseDtkModuleTestNamespace represents test case namespace name.
	UseDtkModuleTestNamespace = "54283-use-dtk"
	// UseLocalMultiStageTestNamespace represents test case namespace name.
	UseLocalMultiStageTestNamespace = "53651-multi-stage"
	// WebhookModuleTestNamespace represents test case namespace name.
	WebhookModuleTestNamespace = "webhook"
	// SimpleKmodModuleTestNamespace represents test case namespace name.
	SimpleKmodModuleTestNamespace = "simple-kmod"
	// DevicePluginTestNamespace represents test case namespace name.
	DevicePluginTestNamespace = "53678-devplug"
	// RealtimeKernelNamespace represents test case namespace name.
	RealtimeKernelNamespace = "53656-rtkernel"
	// FirmwareTestNamespace represents test case namespace name.
	FirmwareTestNamespace = "simple-kmod-firmware"
	// ModuleBuildAndSignNamespace represents test case namespace name.
	ModuleBuildAndSignNamespace = "56252"
	// InTreeReplacementNamespace represents test case namespace name.
	InTreeReplacementNamespace = "62745"
	// MultipleModuleTestNamespace represents test case namespace name.
	MultipleModuleTestNamespace = "multiple-modules"
	// VersionModuleTestNamespace represents test case namespace name.
	VersionModuleTestNamespace = "modver"
	// PreflightTemplateImage represents image against which preflightvalidationocp will build against.
	PreflightTemplateImage = "quay.io/openshift-release-dev/ocp-release:4.10.15-%s"
	// PreflightName represents preflightvalidation ocp object name.
	PreflightName = "preflight"
	// ScannerTestNamespace represents test case namespace name.
	ScannerTestNamespace = "kmm-scanner"
	// ReasonBuildCompleted represents event reason for a build completed.
	ReasonBuildCompleted = "BuildCompleted"
	// ReasonBuildCreated represents event reason for a build created.
	ReasonBuildCreated = "BuildCreated"
	// ReasonBuildStarted represents event reason for a build started.
	ReasonBuildStarted = "BuildStarted"
	// ReasonBuildSucceeded represents event reason for a build succeeded.
	ReasonBuildSucceeded = "BuildSucceeded"
	// ReasonSignCreated represents event reason for a sign created.
	ReasonSignCreated = "SignCreated"
	// ReasonSignSucceeded represents event reason for a sign succeeded.
	ReasonSignSucceeded = "SignSucceeded"
	// ReasonModuleLoaded represents event reason for a module loaded.
	ReasonModuleLoaded = "ModuleLoaded"
	// ReasonModuleUnloaded represents event reason for a module unloaded.
	ReasonModuleUnloaded = "ModuleUnloaded"
)
