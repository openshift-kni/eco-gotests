package accelparams

import (
	"fmt"
	"slices"
	"time"

	"github.com/openshift-kni/k8sreporter"
	corev1 "k8s.io/api/core/v1"
)

var (
	// IbuWorkloadImage is the test workload image.
	IbuWorkloadImage = `registry.redhat.io/openshift4/ose-hello-openshift-
	rhel8@sha256:10dca31348f07e1bfb56ee93c324525cceefe27cb7076b23e42ac181e4d1863e`
	// UpgradeChannel is the desired upgrade channel.
	UpgradeChannel = "stable-4.14"
	// DeploymentName is the name of the test workload.
	DeploymentName = "test-workload"
	// DefaultTimeout represents the default timeout for most of Eventually/PollImmediate functions.
	DefaultTimeout = 300 * time.Second
	// ContainerCmdBash start a container command with bash.
	ContainerCmdBash = []string{"/bin/bash", "-c"}
	// ContainerCmdSleep start a container with a sleep, later execute the needed
	// command in the container.
	ContainerCmdSleep = append(slices.Clone(ContainerCmdBash), "sleep infinity")
	// TestNamespaceName is the namespace where the workload is deployed.
	TestNamespaceName = "test-ns"
	// containerLabelKey is the test container label key.
	containerLabelKey = "app"
	// containerLabelVal is the test container label value.
	containerLabelVal = "test-workload"
	// ContainerLabelsMap labels in an map used when creating the workload container.
	ContainerLabelsMap = map[string]string{containerLabelKey: containerLabelVal}
	// ContainerLabelsStr labels in a str used when creating the workload container.
	ContainerLabelsStr = fmt.Sprintf("%s=%s", containerLabelKey, containerLabelVal)
	// ServicePort is the workload service port.
	ServicePort int32 = 8080
	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{"accel-test": "accel-test"}
	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.PodList{}},
	}
)
