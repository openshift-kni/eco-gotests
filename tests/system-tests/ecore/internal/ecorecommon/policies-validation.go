package ecorecommon

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

// ValidateAllPoliciesCompliant asserts all policies are compliant.
func ValidateAllPoliciesCompliant() {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Check all policies are compliant")

	var (
		allPolicies []*ocm.PolicyBuilder
		err         error
		ctx         SpecContext
	)

	Eventually(func() bool {
		allPolicies, err = ocm.ListPoliciesInAllNamespaces(APIClient,
			runtimeclient.ListOptions{Namespace: ECoreConfig.PolicyNS})

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Error listing policies: %v", err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Found %d policies", len(allPolicies))

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		"Failed to list policies")

	Expect(len(allPolicies)).NotTo(Equal(0), "Found 0 policies")

	for _, policy := range allPolicies {
		pName := policy.Definition.Name
		glog.V(ecoreparams.ECoreLogLevel).Infof("Processing policy is %q", pName)

		pState := string(policy.Object.Status.ComplianceState)

		glog.V(ecoreparams.ECoreLogLevel).Infof("Policy %q is in %q state", pName, pState)

		Expect(pState).To(Equal("Compliant"),
			fmt.Sprintf("NonCompliant: Policy %q is in %q state", pName, pState))
	}
}
