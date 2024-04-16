package tests

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/scc"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var _ = Describe(
	"Verify scc activation succeeded",
	Ordered,
	ContinueOnFailure,
	Label(vcoreparams.Label), func() {
		It("Verify scc activation", reportxml.ID("60042"),
			Label(vcoreparams.LabelVCoreDeployment), func() {
				By("Get available control-plane-worker nodes")
				nodesList, err := nodes.List(APIClient, VCoreConfig.VCoreCpLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get control-plane-worker nodes list; %s", err)
				Expect(len(nodesList)).ToNot(Equal(0), "control-plane-worker nodes list is empty")

				sccBuilder := scc.NewBuilder(APIClient, vcoreparams.SccName, "RunAsAny", "MustRunAs").
					WithHostDirVolumePlugin(true).
					WithHostIPC(false).
					WithHostNetwork(false).
					WithHostPID(false).
					WithHostPorts(false).
					WithPrivilegedEscalation(true).
					WithPrivilegedContainer(true).
					WithAllowCapabilities(vcoreparams.CpSccAllowCapabilities).
					WithFSGroup("MustRunAs").
					WithFSGroupRange(1000, 1000).
					WithGroups(vcoreparams.CpSccGroups).
					WithPriority(nil).
					WithReadOnlyRootFilesystem(false).
					WithDropCapabilities(vcoreparams.CpSccDropCapabilities).
					WithSupplementalGroups("RunAsAny").
					WithVolumes(vcoreparams.CpSccVolumes)

				if !sccBuilder.Exists() {
					glog.V(100).Infof("Create securityContextConstraints instance")
					scc, err := sccBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create %s scc instance; %s",
						vcoreparams.SccName, err)
					Expect(scc.Exists()).To(Equal(true),
						"Failed to create %s SCC", vcoreparams.SccName)
				}
			})
	})
