package tests

import (
	"fmt"
	"strings"
	"time"

	orderedMap "github.com/wk8/go-ordered-map/v2"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
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
	// firstVlanId represents vlan number which is used for the first vlan multus interface.
	firstVlanID = 100
	// secondVlanID represents vlan number which is used for the second vlan multus interface.
	secondVlanID = 200
	// firstTapInterfaceName represents the name of the first tap interface.
	firstTapInterfaceName = "ext0"
	// secondTapInterfaceName represents the name of the second tap interface.
	secondTapInterfaceName = "ext1"
	// interfaceBasedOnFirstTap represents the name of the interface which is based on top of first tap ext0.
	interfaceBasedOnFirstTap = "ext0-1"
	// secondInterfaceBasedOnFirstTap  represents the name of the second interface which is based on top of
	// second tap ext0.
	secondInterfaceBasedOnFirstTap = "ext0-2"
	// interfaceBasedOnSecondTap represents the name of the interface which is based on top of second tap ext1.
	interfaceBasedOnSecondTap = "ext1-1"
	// secondInterfaceBasedOnSecondTap represents the name of the second interface which
	// is based on top of second tap ext1.
	secondInterfaceBasedOnSecondTap = "ext1-2"
	// firstNetIPv6 represents IPv6Address which is used for the first interface.
	firstNetIPv6 = "2001:100::1/64"
	// secondNetIPv6 represents IPv6Address which is used for the second interface.
	secondNetIPv6 = "2001:101::1/64"
	// firstNetIPv4 represents IPv4Address which is used for the first interface.
	firstNetIPv4 = "192.168.100.1/24"
	// secondNetIPv4 represents IPv4Address which is used for the second interface.
	secondNetIPv4 = "192.168.200.1/24"
	// tapOneNadName represents the name of the first tap NetworkAttachmentDefinition.
	tapOneNadName = "tap-one"
	// tapOneNadName represents the name of the second tap NetworkAttachmentDefinition.
	tapTwoNadName = "tap-two"
	// vlanOneNadName represents the name of the first vlan NetworkAttachmentDefinition.
	vlanOneNadName = "vlan-one"
	// vlanOneNadName represents the name of the second vlan NetworkAttachmentDefinition.
	vlanTwoNadName = "vlan-two"
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
	trueFlag  = true
	falseFlag = false
	defaultSC = &corev1.SecurityContext{
		AllowPrivilegeEscalation: &falseFlag,
		RunAsNonRoot:             &trueFlag,
		SeccompProfile: &corev1.SeccompProfile{
			Type: "RuntimeDefault",
		},
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
				tapOne := defineAndCreateTapNad(tapOneNadName, 0, 0, enabledSysctlFlags)

				By("Creating tap-two NetworkAttachmentDefinition")
				tapTwo := defineAndCreateTapNad(tapTwoNadName, customUserID, customGroupID, disabledSysctlFlags)

				By("Creating mac vlan one NetworkAttachmentDefinition")
				macVlanOne := defineAndCreateMacVlanNad("mac-vlan-one", firstTapInterfaceName, nad.IPAMStatic())

				By("Creating mac vlan two NetworkAttachmentDefinition")
				macVlanTwo := defineAndCreateMacVlanNad("mac-vlan-two", secondTapInterfaceName, nad.IPAMStatic())

				By("Setting pod network annotation")
				macVlanOnePodInterfaceDefinition := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					macVlanOne.Definition.Name, tsparams.TestNamespaceName, interfaceBasedOnFirstTap, []string{firstNetIPv6})
				macVlanTwoPodInterfaceDefinition := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					macVlanTwo.Definition.Name, tsparams.TestNamespaceName, interfaceBasedOnSecondTap, []string{secondNetIPv6})
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
				doesInterfaceHasCorrectMasterAndIPAddress(runningPod, interfaceBasedOnFirstTap, firstNetIPv6)
				doesInterfaceHasCorrectMasterAndIPAddress(runningPod, interfaceBasedOnSecondTap, secondNetIPv6)

				By("Verifying that devises have correct interface type")
				doesMacVlanHasCorrectConfig(runningPod, interfaceBasedOnFirstTap, firstTapInterfaceName)
				doesMacVlanHasCorrectConfig(runningPod, interfaceBasedOnSecondTap, secondTapInterfaceName)
			})

			It("VLANs interfaces and dual-stack static IPAM", polarion.ID("63734"), func() {
				By("Creating tap-one NetworkAttachmentDefinition")
				tapOne := defineAndCreateTapNad(tapOneNadName, 0, 0, enabledSysctlFlags)

				By("Creating tap-two NetworkAttachmentDefinition")
				tapTwo := defineAndCreateTapNad(tapTwoNadName, customUserID, customGroupID, disabledSysctlFlags)

				By("Creating vlan one NetworkAttachmentDefinition")
				vlanOne := defineAndCreateVlanNad(vlanOneNadName, firstTapInterfaceName, firstVlanID, nad.IPAMStatic())

				By("Creating vlan two NetworkAttachmentDefinition")
				vlanTwo := defineAndCreateVlanNad(vlanTwoNadName, secondTapInterfaceName, secondVlanID, nad.IPAMStatic())

				By("Setting pod network annotation")
				vlanOnePodInterfaceDefinition := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					vlanOne.Definition.Name,
					tsparams.TestNamespaceName,
					interfaceBasedOnFirstTap,
					[]string{firstNetIPv4, firstNetIPv6})
				vlanTwoPodInterfaceDefinition := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					vlanTwo.Definition.Name,
					tsparams.TestNamespaceName,
					interfaceBasedOnSecondTap,
					[]string{secondNetIPv4, secondNetIPv6})
				podNetCfg := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					tapOne.Definition.Name, tsparams.TestNamespaceName, firstTapInterfaceName, nil)
				podNetCfg = append(podNetCfg, pod.StaticIPAnnotationWithInterfaceAndNamespace(
					tapTwo.Definition.Name, tsparams.TestNamespaceName, secondTapInterfaceName, nil)...)
				podNetCfg = append(podNetCfg, vlanOnePodInterfaceDefinition...)
				podNetCfg = append(podNetCfg, vlanTwoPodInterfaceDefinition...)

				By("Creating test pod and wait util it's running")
				runningPod, err := pod.NewBuilder(
					APIClient,
					"pod-one",
					tsparams.TestNamespaceName,
					NetConfig.CnfNetTestContainer).
					WithSecondaryNetwork(podNetCfg).
					CreateAndWaitUntilRunning(tsparams.DefaultTimeout)
				Expect(err).ToNot(HaveOccurred(), "Fail to create test pod")

				By("Verifying that device has correct tun type and user/group")
				doesTapHasCorrectConfig(runningPod, firstTapInterfaceName, 0, 0)
				doesTapHasCorrectConfig(runningPod, secondTapInterfaceName, customUserID, customGroupID)

				By("Verifying that devices have correct sysctl flags")
				verifySysctlKernelParametersConfiguredOnPodInterface(
					runningPod, enabledSysctlFlags, firstTapInterfaceName)
				verifySysctlKernelParametersConfiguredOnPodInterface(
					runningPod, disabledSysctlFlags, secondTapInterfaceName)

				By("Verifying that devices have correct ipv4 address")
				doesInterfaceHasCorrectMasterAndIPAddress(runningPod, interfaceBasedOnFirstTap, firstNetIPv4)
				doesInterfaceHasCorrectMasterAndIPAddress(runningPod, interfaceBasedOnSecondTap, secondNetIPv4)

				By("Verifying that devices have correct ipv6 address")
				doesInterfaceHasCorrectMasterAndIPAddress(runningPod, interfaceBasedOnFirstTap, firstNetIPv6)
				doesInterfaceHasCorrectMasterAndIPAddress(runningPod, interfaceBasedOnSecondTap, secondNetIPv6)

				By("Verifying that devises have correct interface type")
				doesVlanHasCorrectConfig(runningPod, interfaceBasedOnFirstTap, firstTapInterfaceName, firstVlanID)
				doesVlanHasCorrectConfig(runningPod, interfaceBasedOnSecondTap, secondTapInterfaceName, secondVlanID)
			})

			It("two IP-VLAN and two VLAN interfaces, IPAM dual-stack whereabout, Pod restart using deployment",
				polarion.ID("63735"), func() {
					nadNamesInterfaceNamesMap := orderedMap.New[string, string]()

					By("Creating tap-one NetworkAttachmentDefinition")
					tapOne := defineAndCreateTapNad(tapOneNadName, 0, 0, enabledSysctlFlags)
					nadNamesInterfaceNamesMap.Set(tapOne.Definition.Name, firstTapInterfaceName)

					By("Creating tap-two NetworkAttachmentDefinition")
					tapTwo := defineAndCreateTapNad(tapTwoNadName, customUserID, customGroupID, disabledSysctlFlags)
					nadNamesInterfaceNamesMap.Set(tapTwo.Definition.Name, secondTapInterfaceName)

					By("Creating vlan one NetworkAttachmentDefinition")
					whereaboutNetOne := nad.WhereAboutsAppendRange(
						nad.IPAMWhereAbouts(firstNetIPv4, "192.168.100.254"), firstNetIPv6, "2001:100::10")
					vlanOne := defineAndCreateVlanNad(vlanOneNadName, firstTapInterfaceName, firstVlanID, whereaboutNetOne)
					nadNamesInterfaceNamesMap.Set(vlanOne.Definition.Name, interfaceBasedOnFirstTap)

					By("Creating vlan two NetworkAttachmentDefinition")
					whereaboutNetTwo := nad.WhereAboutsAppendRange(
						nad.IPAMWhereAbouts(secondNetIPv4, "192.168.200.254"), secondNetIPv6, "2001:101::10")
					vlanTwo := defineAndCreateVlanNad(vlanTwoNadName, secondTapInterfaceName, secondVlanID, whereaboutNetTwo)
					nadNamesInterfaceNamesMap.Set(vlanTwo.Definition.Name, secondInterfaceBasedOnSecondTap)

					By("Creating ip vlan one NetworkAttachmentDefinition")
					whereaboutNetOne = nad.WhereAboutsAppendRange(
						nad.IPAMWhereAbouts("192.168.110.0/24", "192.168.110.254"), "2001:110::0/64", "2001:110::10")
					ipVlanOne := defineAndCreateIPVlanNad("ip-vlan-one", firstTapInterfaceName, whereaboutNetOne)
					nadNamesInterfaceNamesMap.Set(ipVlanOne.Definition.Name, secondInterfaceBasedOnFirstTap)

					By("Creating ip vlan two NetworkAttachmentDefinition")
					whereaboutNetTwo = nad.WhereAboutsAppendRange(
						nad.IPAMWhereAbouts("192.168.210.0/24", "192.168.210.254"), "2001:210::0/64", "2001:210::10")
					ipVlanTwo := defineAndCreateIPVlanNad("ip-vlan-two", secondTapInterfaceName, whereaboutNetTwo)
					nadNamesInterfaceNamesMap.Set(ipVlanTwo.Definition.Name, interfaceBasedOnSecondTap)

					By("Creating Test deployment")
					deploymentContainer, err := pod.NewContainerBuilder(
						"test", NetConfig.CnfNetTestContainer, []string{"/bin/bash", "-c", "sleep INF"}).
						WithSecurityContext(defaultSC).GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Fail to collect container configuration")
					deploymentBuilder := deployment.NewBuilder(
						APIClient, "deployment-one",
						tsparams.TestNamespaceName, map[string]string{"test": "tap"}, deploymentContainer)

					var deploymentNetCfg []*types.NetworkSelectionElement

					for pair := nadNamesInterfaceNamesMap.Oldest(); pair != nil; pair = pair.Next() {
						deploymentNetCfg = append(deploymentNetCfg, pod.StaticIPAnnotationWithInterfaceAndNamespace(
							pair.Key, tsparams.TestNamespaceName, pair.Value, nil)...,
						)
					}

					_, err = deploymentBuilder.WithSecondaryNetwork(deploymentNetCfg).
						CreateAndWaitUntilReady(tsparams.DefaultTimeout)
					Expect(err).ToNot(HaveOccurred(), "Fail to create deployment")

					By("Collecting deployment pods")
					deploymentPod := fetchNewDeploymentPod()
					testDualStackNetConfigWithTwoTapsTwoIPVLANsTwoVLANsOnTopOfDeploymentWithWhereabouts(deploymentPod)

					By("Removing deployment pod")
					_, err = deploymentPod.DeleteAndWait(tsparams.DefaultTimeout)
					Expect(err).ToNot(HaveOccurred(), "Fail to delete deployment pod")

					By("Collecting restated deployment pods")
					restartedDeploymentPod := fetchNewDeploymentPod()

					By("Re-testing after pod restart")
					testDualStackNetConfigWithTwoTapsTwoIPVLANsTwoVLANsOnTopOfDeploymentWithWhereabouts(restartedDeploymentPod)
				})
		})

		AfterEach(func() {
			By("Cleaning configuration after test")
			cniNs, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull test namespace")
			err = cniNs.CleanObjects(tsparams.DefaultTimeout, pod.GetGVR(), nad.GetGVR(), deployment.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Fail to clean up test namespace")
		})

		AfterAll(func() {
			By("Setting selinux flag container_use_devices to 0 on all compute nodes")
			err := cluster.ExecCmd(APIClient, NetConfig.WorkerLabel, setSEBool+"0")
			Expect(err).ToNot(HaveOccurred(), "Fail to disable selinux flag")
		})
	})

func testDualStackNetConfigWithTwoTapsTwoIPVLANsTwoVLANsOnTopOfDeploymentWithWhereabouts(deploymentPod *pod.Builder) {
	By("Verifying that first tap has correct tun type and user/group")
	doesTapHasCorrectConfig(deploymentPod, firstTapInterfaceName, 0, 0)

	By("Verifying that first tap interface have correct sysctl flags")
	verifySysctlKernelParametersConfiguredOnPodInterface(deploymentPod, enabledSysctlFlags, firstTapInterfaceName)

	By("Verifying that the first vlan device has correct vlanID and ip addresses")
	doesVlanHasCorrectConfig(deploymentPod, interfaceBasedOnFirstTap, firstTapInterfaceName, firstVlanID)
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, interfaceBasedOnFirstTap, "192.168.100.")
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, interfaceBasedOnFirstTap, "2001:100::")

	By("Verifying that the first IPVlan device has correct device type and ip addresses")
	doesIPVlanHasCorrectConfig(deploymentPod, secondInterfaceBasedOnFirstTap, firstTapInterfaceName)
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, secondInterfaceBasedOnFirstTap, "192.168.110.")
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, secondInterfaceBasedOnFirstTap, "2001:110::")

	By("Verifying that second tap has correct tun type and user/group")
	doesTapHasCorrectConfig(deploymentPod, secondTapInterfaceName, customUserID, customGroupID)

	By("Verifying that the second tap interface have correct sysctl flags")
	verifySysctlKernelParametersConfiguredOnPodInterface(deploymentPod, disabledSysctlFlags, secondTapInterfaceName)

	By("Verifying that the second vlan device has correct vlanID and ip addresses")
	doesVlanHasCorrectConfig(deploymentPod, secondInterfaceBasedOnSecondTap, secondTapInterfaceName, secondVlanID)
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, secondInterfaceBasedOnSecondTap, "192.168.200.")
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, secondInterfaceBasedOnSecondTap, "2001:101::")

	By("Verifying that the second IPVlan device has correct device type and ip addresses")
	doesIPVlanHasCorrectConfig(deploymentPod, interfaceBasedOnSecondTap, secondTapInterfaceName)
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, interfaceBasedOnSecondTap, "192.168.210.")
	doesInterfaceHasCorrectMasterAndIPAddress(deploymentPod, interfaceBasedOnSecondTap, "2001:210::")
}

func fetchNewDeploymentPod() *pod.Builder {
	By("Re-Collecting deployment pods")

	var deploymentPodList []*pod.Builder

	Eventually(func() bool {
		deploymentPodList, _ = pod.List(APIClient, tsparams.TestNamespaceName, v1.ListOptions{})

		return len(deploymentPodList) == 1

	}, time.Minute, tsparams.DefaultTimeout).Should(BeTrue(), "Failed to collect deployment pods")

	err := deploymentPodList[0].WaitUntilRunning(tsparams.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Fail to get pod running state")

	return deploymentPodList[0]
}

func doesInterfaceHasCorrectMasterAndIPAddress(podObject *pod.Builder, intName, ipAddr string) {
	buffer, err := podObject.ExecCommand([]string{"ip", "addr", "show", intName})
	Expect(err).ToNot(HaveOccurred(), "Fail to get interface ip address on pod")
	Expect(strings.Contains(buffer.String(), ipAddr)).To(BeTrue(), fmt.Sprintf("Fail to detect requested ip %s", ipAddr))
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

func doesVlanHasCorrectConfig(podObject *pod.Builder, intName, masterIntName string, vlanID uint16) {
	interfaceConfig := collectLinkConfigFromPod(podObject, intName)
	for _, expectedPattern := range []string{
		fmt.Sprintf("vlan protocol 802.1Q id %d", vlanID),
		fmt.Sprintf("%s@%s", intName, masterIntName)} {
		Expect(strings.Contains(interfaceConfig, expectedPattern)).To(BeTrue(), "Fail to find required config")
	}
}

func doesIPVlanHasCorrectConfig(podObject *pod.Builder, intName, masterIntName string) {
	interfaceConfig := collectLinkConfigFromPod(podObject, intName)
	for _, expectedPattern := range []string{"ipvlan  mode l2 bridge", fmt.Sprintf("%s@%s", intName, masterIntName)} {
		Expect(strings.Contains(interfaceConfig, expectedPattern)).To(BeTrue(), "Fail to find required config")
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

func defineAndCreateVlanNad(name, intName string, vlanID uint16, ipam *nad.IPAM) *nad.Builder {
	masterPlugin, err := nad.NewMasterVlanPlugin(name, vlanID).WithMasterInterface(intName).
		WithIPAM(ipam).WithLinkInContainer().GetMasterPluginConfig()
	Expect(err).ToNot(HaveOccurred(), "Fail to set MasterVlan plugin")

	return defineAndCreateNad(name, masterPlugin)
}

func defineAndCreateIPVlanNad(name, intName string, ipam *nad.IPAM) *nad.Builder {
	masterPlugin, err := nad.NewMasterIPVlanPlugin(name).WithMasterInterface(intName).WithIPAM(ipam).
		WithLinkInContainer().GetMasterPluginConfig()
	Expect(err).ToNot(HaveOccurred(), "Fail to set IPVlan plugin")

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
