package tests

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/scc"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsunginittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsungparams"
)

var _ = Describe(
	"Verify scc activation succeeded",
	Ordered,
	ContinueOnFailure,
	Label(samsungparams.Label), func() {
		It("Verify scc activation", polarion.ID("60042"),
			Label("samsungvcoredeployment"), func() {

				By("Get available samsung-cnf nodes")
				nodesList, err := nodes.List(APIClient, SamsungConfig.SamsungCnfLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get samsung-cnf nodes list; %s", err)
				Expect(len(nodesList)).ToNot(Equal(0), "samsung-cnf nodes list is empty")

				sccBuilder := scc.NewBuilder(APIClient, samsungparams.SccName, "RunAsAny", "MustRunAs").
					WithHostDirVolumePlugin(true).
					WithHostIPC(false).
					WithHostNetwork(false).
					WithHostPID(false).
					WithHostPorts(false).
					WithPrivilegedEscalation(true).
					WithPrivilegedContainer(true).
					WithAllowCapabilities(samsungparams.CnfSccAllowCapabilities).
					WithFSGroup("MustRunAs").
					WithFSGroupRange(1000, 1000).
					WithGroups(samsungparams.CnfSccGroups).
					WithPriority(nil).
					WithReadOnlyRootFilesystem(false).
					WithDropCapabilities(samsungparams.CnfSccDropCapabilities).
					WithSupplementalGroups("RunAsAny").
					WithVolumes(samsungparams.CnfSccVolumes)

				if !sccBuilder.Exists() {
					glog.V(100).Infof("Create securityContextConstraints instance")
					scc, err := sccBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create %s scc instance; %s",
						samsungparams.SccName, err)
					Expect(scc.Exists()).To(Equal(true),
						"Failed to create %s SCC", samsungparams.SccName)
				}
			})
	})
