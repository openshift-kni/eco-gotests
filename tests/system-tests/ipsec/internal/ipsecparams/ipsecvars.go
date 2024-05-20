package ipsecparams

import (
	"fmt"
	"slices"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsparams"
	"github.com/openshift-kni/k8sreporter"
	corev1 "k8s.io/api/core/v1"
)

var (
	// Iperf3DeploymentName represents the deployment name used for launching iperf3 client and server workloads.
	Iperf3DeploymentName = "iperf3"

	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{systemtestsparams.Label, Label}

	// ReporterNamespacesToDump tells the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"test": "ipsec-test-workload",
	}

	// ReporterCRDsToDump tells the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.PodList{}},
	}

	// TestNamespaceName is used for defining the namespace name where test resources are created.
	TestNamespaceName = "ipsec-system-tests"

	containerLabelKey = "ipsec-systemtest"
	containerLabelVal = "iperf3"
	// ContainerLabelsMap labels in an map used when creating the iperf3 container.
	ContainerLabelsMap = map[string]string{containerLabelKey: containerLabelVal}
	// ContainerLabelsStr labels in a str used when creating the iperf3 container.
	ContainerLabelsStr = fmt.Sprintf("%s=%s", containerLabelKey, containerLabelVal)

	// ContainerCmdBash start a container command with bash.
	ContainerCmdBash = []string{"/bin/bash", "-c"}

	// ContainerCmdChroot start a container command with with chroot for root access.
	ContainerCmdChroot = []string{"chroot", "/rootfs", "/bin/sh", "-c"}

	// ContainerCmdSleep start a container with a sleep, later execute the needed
	// command in the container.
	ContainerCmdSleep = append(slices.Clone(ContainerCmdBash), "sleep infinity")

	// IpsecCmdShow IPSec command string to check for active/open tunnels.
	IpsecCmdShow = append(slices.Clone(ContainerCmdChroot), "ipsec show")

	// IpsecCmdTrafficStatus IPSec command string to check for tunnel packets.
	IpsecCmdTrafficStatus = append(slices.Clone(ContainerCmdChroot), "ipsec trafficstatus")

	// Iperf3OptionBind option to bind to an IP.
	Iperf3OptionBind = "-B"

	// Iperf3OptionBytes option to specify how may bytes to send.
	Iperf3OptionBytes = "-n"

	// Iperf3OptionPort option to use a specific port, intead of the default 5201.
	Iperf3OptionPort = "-p"

	// Iperf3ClientBaseCmd Start an iperf3 client with JSON output,
	// need to append the serverIP, port, and bytes.
	Iperf3ClientBaseCmd = []string{"iperf3", "-J", "-c"}

	// Iperf3ServerBaseCmd Start an iperf3 server that stops after 1 iperf3 client
	// connection completes, need to append the server IP to bind to.
	// If the client does not connect in 180 seconds, the server will be interrupted.
	Iperf3ServerBaseCmd = []string{"timeout", "180", "iperf3", "--one-off", "-J", "-s"}
)
