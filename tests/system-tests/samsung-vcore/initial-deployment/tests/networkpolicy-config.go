package tests

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/networkpolicy"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsunginittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsungparams"
)

var _ = Describe(
	"Verify networkpolicy could be configured according to the Samsung requirements",
	Label(samsungparams.Label), func() {
		It("Verify network policy configuration procedure", reportxml.ID("60086"),
			Label("samsungvcoredeployment"), func() {

				for _, npNamespace := range samsungparams.NetworkPoliciesNamespaces {
					nsBuilder := namespace.NewBuilder(APIClient, npNamespace)

					inNamespaceStr := fmt.Sprintf("in namespace %s", npNamespace)

					glog.V(100).Infof("Create %s namespace", npNamespace)
					_, err := nsBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "failed to create namespace %s",
						npNamespace)

					glog.V(100).Infof("Create networkpolicy %s %s",
						samsungparams.MonitoringNetworkPolicyName, inNamespaceStr)

					npBuilder, err := networkpolicy.NewNetworkPolicyBuilder(APIClient,
						samsungparams.MonitoringNetworkPolicyName, npNamespace).WithNamespaceIngressRule(
						samsungparams.NetworkPolicyMonitoringNamespaceSelectorMatchLabels,
						nil).WithPolicyType(samsungparams.NetworkPolicyType).Create()
					Expect(err).ToNot(HaveOccurred(), "failed to create networkpolicy %s %s",
						samsungparams.MonitoringNetworkPolicyName, inNamespaceStr)

					glog.V(100).Infof("Verify networkpolicy %s successfully created %s",
						samsungparams.MonitoringNetworkPolicyName, inNamespaceStr)
					Expect(npBuilder.Exists()).To(Equal(true), "networkpolicy %s not found %s",
						samsungparams.MonitoringNetworkPolicyName, inNamespaceStr)

					glog.V(100).Infof("Create networkpolicy %s %s",
						samsungparams.AllowAllNetworkPolicyName, inNamespaceStr)
					npBuilder, err = networkpolicy.NewNetworkPolicyBuilder(APIClient,
						samsungparams.AllowAllNetworkPolicyName, npNamespace).
						WithPolicyType(samsungparams.NetworkPolicyType).Create()
					Expect(err).ToNot(HaveOccurred(), "failed to create networkpolicy %s objects %s",
						samsungparams.AllowAllNetworkPolicyName, inNamespaceStr)

					glog.V(100).Infof("Verify networkpolicy %s successfully created %s",
						samsungparams.MonitoringNetworkPolicyName, inNamespaceStr)
					Expect(npBuilder.Exists()).To(Equal(true), "networkpolicy %s not found %s",
						samsungparams.AllowAllNetworkPolicyName, inNamespaceStr)

				}
			})
	})
