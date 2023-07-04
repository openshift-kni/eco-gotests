package tests

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/cni/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

const (
	// setSEBool represents cmd which allow to set selinux boolean container_use_devicess.
	setSEBool = "setsebool container_use_devices "
	// customUserID represents custom linux user id.
	customUserID = 1001
	// customGroupID represents custom linux group id.
	customGroupID = 1001
	// firstTapInterfaceName represents the name of the first tap interface.
	firstTapInterfaceName = "ext0"
	// secondTapInterfaceName represents the name of the second tap interface.
	secondTapInterfaceName = "ext1"
	// macVlanOnBasedOnFirstTap represents the name of the mac-vlan interface which is based on top of first tap ext0.
	macVlanOnBasedOnFirstTap = "ext0.1"
	// macVlanOnBasedOnSecondTap represents the name of the mac-vlan interface which is based on top of second tap ext1.
	macVlanOnBasedOnSecondTap = "ext1.2"
	// firstNetIPv6 represents IPv6Address which is used for the first interface.
	firstNetIPv6 = "2001:100::1/64"
	// secondNetIPv6 represents IPv6Address which is used for the second interface.
	secondNetIPv6 = "2001:101::1/64"
)

var (
	// enabledSysctlFlags represents sysctl configuration with few enable flags for sysctl plugin.
	enabledSysctlFlags = map[string]string{
		"net.ipv6.conf.IFNAME.accept_ra":  "1",
		"net.ipv4.conf.IFNAME.arp_accept": "1",
	}
	// disabledSysctlFlags represents sysctl configuration with few disabled flags for sysctl plugin.
	disabledSysctlFlags = map[string]string{
		"net.ipv6.conf.IFNAME.accept_ra":  "0",
		"net.ipv4.conf.IFNAME.arp_accept": "0",
	}
)

var _ = Describe("", Ordered,
	Label(tsparams.LabelTapTestCases), ContinueOnFailure, func() {

		BeforeAll(func() {
			By("Setting selinux flag container_use_devices to 1 on all compute nodes")
			err := cluster.ExecCmd(APIClient, NetConfig.WorkerLabel, setSEBool+"1")
			Expect(err).ToNot(HaveOccurred(), "Fail to enable selinux flag")

		})

		Context("two tap devices plus sysctl with", func() {
			It("MAC-VLANs interfaces and IPv6 static IPAM", polarion.ID("63732"), func() {
				By("Creating tap-one NetworkAttachmentDefinition")
				tapOne := defineAndCreateTapNad("tap-one", 0, 0, enabledSysctlFlags)

				By("Creating tap-two NetworkAttachmentDefinition")
				tapTwo := defineAndCreateTapNad("tap-two", customUserID, customGroupID, disabledSysctlFlags)

				By("Creating mac vlan one NetworkAttachmentDefinition")
				macVlanOne := defineAndCreateMacVlanNad("mac-vlan-one", firstTapInterfaceName, nad.IPAMStatic())

				By("Creating mac vlan two NetworkAttachmentDefinition")
				macVlanTwo := defineAndCreateMacVlanNad("mac-vlan-two", secondTapInterfaceName, nad.IPAMStatic())

				By("Setting pod network annotation")
				macVlanOnePodInterfaceDefinition := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					macVlanOne.Definition.Name, tsparams.TestNamespaceName, macVlanOnBasedOnFirstTap, []string{firstNetIPv6})
				macVlanTwoPodInterfaceDefinition := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					macVlanTwo.Definition.Name, tsparams.TestNamespaceName, macVlanOnBasedOnSecondTap, []string{secondNetIPv6})
				podNetCfg := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					tapOne.Definition.Name, tsparams.TestNamespaceName, firstTapInterfaceName, nil)
				podNetCfg = append(
					podNetCfg, pod.StaticIPAnnotationWithInterfaceAndNamespace(
						tapTwo.Definition.Name, tsparams.TestNamespaceName, secondTapInterfaceName, nil)...)
				podNetCfg = append(podNetCfg, macVlanOnePodInterfaceDefinition...)
				podNetCfg = append(podNetCfg, macVlanTwoPodInterfaceDefinition...)

				By("Creating test pod and wait util it's running")
				runningPod, err := pod.NewBuilder(
					APIClient, "pod-one",
					tsparams.TestNamespaceName,
					NetConfig.CnfNetTestContainer).
					WithSecondaryNetwork(podNetCfg).
					CreateAndWaitUntilRunning(tsparams.DefaultTimeout)
				Expect(err).ToNot(HaveOccurred(), "Fail to create test pod")

				By("Verifying that devices have correct tun type and user/group")
				doesTapHasCorrectConfig(runningPod, firstTapInterfaceName, 0, 0)
				doesTapHasCorrectConfig(runningPod, secondTapInterfaceName, customUserID, customGroupID)

				By("Verifying that devices have correct sysctl flags")
				verifySysctlKernelParametersConfiguredOnPodInterface(runningPod, enabledSysctlFlags, firstTapInterfaceName)
				verifySysctlKernelParametersConfiguredOnPodInterface(runningPod, disabledSysctlFlags, secondTapInterfaceName)

				By("Verifying that devices have correct ip addresses")
				doesInterfaceHasCorrectIPAddress(runningPod, macVlanOnBasedOnFirstTap, firstNetIPv6)
				doesInterfaceHasCorrectIPAddress(runningPod, macVlanOnBasedOnSecondTap, secondNetIPv6)

				By("Verifying that devises have correct interface type")
				doesMacVlanHasCorrectConfig(runningPod, macVlanOnBasedOnFirstTap, firstTapInterfaceName)
				doesMacVlanHasCorrectConfig(runningPod, macVlanOnBasedOnSecondTap, secondTapInterfaceName)
			})

		})

		AfterEach(func() {
			cniNs, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull test namespace")
			err = cniNs.CleanObjects(tsparams.DefaultTimeout, pod.GetGVR(), nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Fail to clean up test namespace")
		})

		AfterAll(func() {
			By("Setting selinux flag container_use_devices to 0 on all compute nodes")
			err := cluster.ExecCmd(APIClient, NetConfig.WorkerLabel, setSEBool+"0")
			Expect(err).ToNot(HaveOccurred(), "Fail to disable selinux flag")
		})
	})

func doesInterfaceHasCorrectIPAddress(podObject *pod.Builder, intName, ipAddr string) {
	buffer, err := podObject.ExecCommand([]string{"ip", "addr", "show", intName})
	Expect(err).ToNot(HaveOccurred(), "Fail to get interface ip address on pod")
	Expect(strings.Contains(buffer.String(), ipAddr)).To(BeTrue(), "Fail to detect requested ip")
}

func collectLinkConfigFromPod(podObject *pod.Builder, intName string) string {
	buffer, err := podObject.ExecCommand([]string{"ip", "-d", "link", "show", "dev", intName})
	Expect(err).ToNot(HaveOccurred(), "Fail to get link information on pod")

	return buffer.String()
}

func doesTapHasCorrectConfig(podObject *pod.Builder, intName string, user, group int) {
	interfaceConfig := collectLinkConfigFromPod(podObject, intName)
	Expect(strings.Contains(interfaceConfig, "tun type tap")).To(BeTrue(),
		"Fail to detect tap interface type")

	if user != 0 {
		Expect(strings.Contains(interfaceConfig, fmt.Sprintf("user %d group %d", user, group))).To(BeTrue(),
			"Fail to detect username and group on interface")
	}
}

func doesMacVlanHasCorrectConfig(podObject *pod.Builder, intName, masterIntName string) {
	interfaceConfig := collectLinkConfigFromPod(podObject, intName)
	for _, expectedPattern := range []string{"macvlan mode bridge", fmt.Sprintf("%s@%s", intName, masterIntName)} {
		Expect(strings.Contains(interfaceConfig, expectedPattern)).To(BeTrue(),
			"Fail to find required config")
	}
}

func defineAndCreateTapNad(name string, user, group int, sysctlConfig map[string]string) *nad.Builder {
	plugins := []nad.Plugin{
		*nad.TapPlugin(user, group, true), *nad.TuningSysctlPlugin(true, sysctlConfig)}
	tap, err := nad.NewBuilder(APIClient, name, tsparams.TestNamespaceName).WithPlugins(name, &plugins).Create()
	Expect(err).ToNot(HaveOccurred(), "Fail to create tap NetworkAttachmentDefinition")

	return tap
}

func defineAndCreateNad(name string, masterPlugin *nad.MasterPlugin) *nad.Builder {
	createdNad, err := nad.NewBuilder(APIClient, name, tsparams.TestNamespaceName).WithMasterPlugin(masterPlugin).Create()
	Expect(err).ToNot(HaveOccurred(), "Fail to create NetworkAttachmentDefinition")

	return createdNad
}

func defineAndCreateMacVlanNad(name, intName string, ipam *nad.IPAM) *nad.Builder {
	masterPlugin, err := nad.NewMasterMacVlanPlugin(name).WithMasterInterface(intName).
		WithIPAM(ipam).WithLinkInContainer().GetMasterPluginConfig()
	Expect(err).ToNot(HaveOccurred(), "Fail to set MasterMacVlan plugin")

	return defineAndCreateNad(name, masterPlugin)
}

func verifySysctlKernelParametersConfiguredOnPodInterface(
	podUnderTest *pod.Builder, sysctlPluginConfig map[string]string, interfaceName string) {
	for key, value := range sysctlPluginConfig {
		sysctlKernelParam := strings.Replace(key, "IFNAME", interfaceName, 1)

		By(fmt.Sprintf("Validate sysctl flag: %s has the right value in pod's interface: %s",
			sysctlKernelParam, interfaceName))

		cmdBuffer, err := podUnderTest.ExecCommand([]string{"sysctl", "-n", sysctlKernelParam})
		Expect(err).ToNot(HaveOccurred(), "Fail to execute cmd command on the pod")
		Expect(strings.TrimSpace(cmdBuffer.String())).To(BeIdenticalTo(value),
			"sysctl kernel param is not in expected state")
	}
}
