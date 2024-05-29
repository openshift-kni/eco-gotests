package vcorecommon

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/cgroup"
	configv1 "github.com/openshift/api/config/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nodesconfig"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

// VerifyCGroupV2IsADefault assert cGroupV2 is a default for the cluster deployment.
func VerifyCGroupV2IsADefault(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify cgroupv2 is a default for the cluster deployment")

	glog.V(vcoreparams.VCoreLogLevel).Infof("Get current cgroup mode configured for the cluster")

	nodesConfigObj, err := nodesconfig.Pull(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get nodes.config 'cluster' object due to %v", err))

	cgroupMode, err := nodesConfigObj.GetCGroupMode()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get cluster cgroup mode due to %v", err))
	Expect(cgroupMode).ToNot(Equal(configv1.CgroupModeV1), "wrong cgroup mode default found, v1 instead of v2")

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify actual cgroup version configured for the nodes")

	nodesList, err := nodes.List(APIClient)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get cluster nodes list due to %v", err))

	for _, node := range nodesList {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Verify actual cgroup version configured for node %s",
			node.Definition.Name)

		currentCgroupMode, err := cgroup.GetNodeLinuxCGroupVersion(APIClient, node.Definition.Name)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to get cgroup mode value for the node %s due to %v",
				node.Definition.Name, err))
		Expect(currentCgroupMode).To(Equal(configv1.CgroupModeV2), fmt.Sprintf("wrong cgroup mode default "+
			"found configured for the node %s; expected cgroupMode is %v, found cgroupMode is %v",
			node.Definition.Name, configv1.CgroupModeV2, currentCgroupMode))
	}
} // func VerifyCGroupV2IsADefault (ctx SpecContext)

// VerifySwitchBetweenCGroupVersions assert that the cluster can be moved to the cgroupv1 and back.
func VerifySwitchBetweenCGroupVersions(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify that the cluster can be moved to the cgroupv1 and back")

	err := cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV1)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
		configv1.CgroupModeV1, err))

	glog.V(vcoreparams.VCoreLogLevel).Infof("The short sleep to update new values before the following change.")

	time.Sleep(2 * time.Minute)

	err = cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV2)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
		configv1.CgroupModeV2, err))
} // func VerifySwitchBetweenCGroupVersions (ctx SpecContext)

// VerifyCGroupDefault container that contains tests for cgroup verification.
func VerifyCGroupDefault() {
	Describe(
		"cgroup verification",
		Label(vcoreparams.LabelVCoreDeployment), func() {
			BeforeAll(func() {
				By("Check that the current cluster version is greater or equal to the 4.15")

				isGreaterOrEqual, err := platform.CompareOCPVersionWithCurrent(APIClient,
					"4.15",
					true,
					true)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to compare versions due to %v", err))

				if isGreaterOrEqual {
					currentOCPVersion, err := platform.GetOCPVersion(APIClient)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get OCP version due to %v", err))

					Skip(fmt.Sprintf("current OCP version %s is not greater or equal to the 4.15",
						currentOCPVersion))
				}

				By("Insure cgroupv2 configured for the cluster")

				err = cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV2)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
						configv1.CgroupModeV2, err))
			})

			It("Verifies cgroupv2 is a default for the cluster deployment",
				Label("cgroupv2"), reportxml.ID("73370"), VerifyCGroupV2IsADefault)

			It("Verifies that the cluster can be moved to the cgroupv1 and back",
				Label("cgroupv2"), reportxml.ID("73371"), VerifySwitchBetweenCGroupVersions)

			AfterAll(func() {
				By("Restore cgroupv2 cluster configuration")

				err := cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV2)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
						configv1.CgroupModeV2, err))
			})
		})
}
