package ran_du_system_test

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuparams"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe(
	"ZTPPoliciesCompliance",
	Label("ZTPPoliciesCompliance"),
	Ordered,
	ContinueOnFailure,
	func() {
		var (
			clusterName string
			err         error
		)
		BeforeAll(func() {
			By("Fetching Cluster name")
			clusterName, err = platform.GetOCPClusterName(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster name")
		})

		It("Asserts all policies are compliant", func() {
			glog.V(randuparams.RanDuLogLevel).Infof("Check all policies are compliant")
			allPolicies, err := ocm.ListPoliciesInAllNamespaces(APIClient,
				runtimeclient.ListOptions{Namespace: clusterName})
			Expect(err).ToNot(HaveOccurred(), "Failed to get policies in %q NS", clusterName)
			for _, policy := range allPolicies {
				pName := policy.Definition.Name
				glog.V(randuparams.RanDuLogLevel).Infof("Processing policy is %q", pName)

				pState := string(policy.Object.Status.ComplianceState)
				Expect(pState).To(Equal("Compliant"))
				glog.V(randuparams.RanDuLogLevel).Infof("Policy %q is in %q state", pName, pState)
			}

		})
	},
)
