package ecore_system_test

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore Governance Policies",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidatePolicies), func() {
		It("Asserts all policies are compliant", func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Check all policies are compliant")
			allPolicies, err := ocm.ListPoliciesInAllNamespaces(APIClient,
				runtimeclient.ListOptions{Namespace: ECoreConfig.PolicyNS})
			Expect(err).ToNot(HaveOccurred(), "Failed to get policies in ecore NS")
			for _, policy := range allPolicies {
				pName := policy.Definition.Name
				glog.V(ecoreparams.ECoreLogLevel).Infof("Processing policy is %q", pName)

				pState := string(policy.Object.Status.ComplianceState)
				Expect(pState).To(Equal("Compliant"))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Policy %q is in %q state", pName, pState)
			}

		})
	})
