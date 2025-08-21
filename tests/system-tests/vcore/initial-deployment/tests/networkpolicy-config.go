package tests

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/networkpolicy"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var _ = Describe(
	"Verify networkpolicy could be configured according to the vCore requirements",
	Label(vcoreparams.Label), func() {
		It("Verify network policy configuration procedure", reportxml.ID("60086"),
			Label(vcoreparams.LabelVCoreDeployment), func() {

				for _, npNamespace := range vcoreparams.NetworkPoliciesNamespaces {
					nsBuilder := namespace.NewBuilder(APIClient, npNamespace)

					inNamespaceStr := fmt.Sprintf("in namespace %s", npNamespace)

					glog.V(100).Infof("Create %s namespace", npNamespace)
					_, err := nsBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "failed to create namespace %s",
						npNamespace)

					glog.V(100).Infof("Create networkpolicy %s %s",
						vcoreparams.MonitoringNetworkPolicyName, inNamespaceStr)

					npBuilder, err := networkpolicy.NewNetworkPolicyBuilder(APIClient,
						vcoreparams.MonitoringNetworkPolicyName, npNamespace).WithNamespaceIngressRule(
						vcoreparams.NetworkPolicyMonitoringNamespaceSelectorMatchLabels,
						nil).WithPolicyType(vcoreparams.NetworkPolicyType).Create()
					Expect(err).ToNot(HaveOccurred(), "failed to create networkpolicy %s %s",
						vcoreparams.MonitoringNetworkPolicyName, inNamespaceStr)

					glog.V(100).Infof("Verify networkpolicy %s successfully created %s",
						vcoreparams.MonitoringNetworkPolicyName, inNamespaceStr)
					Expect(npBuilder.Exists()).To(Equal(true), "networkpolicy %s not found %s",
						vcoreparams.MonitoringNetworkPolicyName, inNamespaceStr)

					glog.V(100).Infof("Create networkpolicy %s %s",
						vcoreparams.AllowAllNetworkPolicyName, inNamespaceStr)
					npBuilder, err = networkpolicy.NewNetworkPolicyBuilder(APIClient,
						vcoreparams.AllowAllNetworkPolicyName, npNamespace).
						WithPolicyType(vcoreparams.NetworkPolicyType).Create()
					Expect(err).ToNot(HaveOccurred(), "failed to create networkpolicy %s objects %s",
						vcoreparams.AllowAllNetworkPolicyName, inNamespaceStr)

					glog.V(100).Infof("Verify networkpolicy %s successfully created %s",
						vcoreparams.MonitoringNetworkPolicyName, inNamespaceStr)
					Expect(npBuilder.Exists()).To(Equal(true), "networkpolicy %s not found %s",
						vcoreparams.AllowAllNetworkPolicyName, inNamespaceStr)

				}
			})
	})
