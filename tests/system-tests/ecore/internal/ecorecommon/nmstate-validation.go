package ecorecommon

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

// VerifyNMStateInstanceExists assert NMState instace exists.
func VerifyNMStateInstanceExists() {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify NMState instance exists")

	var ctx SpecContext

	Eventually(func() bool {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling in NMState instance %q",
			ecoreparams.NMStateInstanceName)

		nmInstance, err := nmstate.PullNMstate(APIClient, ecoreparams.NMStateInstanceName)

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Error pulling in NMState instance: %v", err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Pulled in NMState instance: %q", nmInstance.Definition.Name)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		"Failed to pull in NMState instance")

	glog.V(ecoreparams.ECoreLogLevel).Infof(
		fmt.Sprintf("Verify all pods in namespace %q are running", ECoreConfig.NMStateOperatorNamespace))

	podsRunning, err := pod.WaitForAllPodsInNamespaceRunning(APIClient,
		ECoreConfig.NMStateOperatorNamespace, 3*time.Minute, metav1.ListOptions{})

	Expect(podsRunning).To(BeTrue(),
		fmt.Sprintf("Some pods in %q namespace ain't running", ECoreConfig.NMStateOperatorNamespace))
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Error checking pods in %s namespace", ECoreConfig.NMStateOperatorNamespace))
}

// VerifyNMStatePoliciesAvailable asserts all NetworkNodePolicies are Available.
func VerifyNMStatePoliciesAvailable() {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify NodeNetworkConfigurationPolicies are Available")

	const ConditionTypeTrue = "True"

	var (
		ctx   SpecContext
		nncps []*nmstate.PolicyBuilder
		err   error
	)

	Eventually(func() bool {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Listing NMState Policies")

		nncps, err = nmstate.ListPolicy(APIClient, goclient.ListOptions{})

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Error listing NMState policies: %v", err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Found %d NMState policies", len(nncps))

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		"Failed listing NMState policies")

	Expect(len(nncps)).ToNot(Equal(0), "0 NodeNetworkConfigurationPolicies found")

	nonAvailableNNCP := make(map[string]string)
	progressingNNCP := make(map[string]string)
	degradedNNCP := make(map[string]string)

	for _, nncp := range nncps {
		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("\t *** Processing %s NodeNetworkConfigurationPolicy ***", nncp.Definition.Name))

		for _, condition := range nncp.Object.Status.Conditions {
			//nolint:nolintlint
			switch condition.Type { //nolint:exhaustive
			//nolint:goconst
			case "Available":
				if condition.Status != ConditionTypeTrue {
					nonAvailableNNCP[nncp.Definition.Name] = condition.Message
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is not Available: %s\n", nncp.Definition.Name, condition.Message))
				} else {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Available: %s\n", nncp.Definition.Name, condition.Message))
				}
			case "Degraded":
				if condition.Status == ConditionTypeTrue {
					degradedNNCP[nncp.Definition.Name] = condition.Message
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Degraded: %s\n", nncp.Definition.Name, condition.Message))
				} else {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Not-Degraded\n", nncp.Definition.Name))
				}
			case "Progressing":
				if condition.Status == ConditionTypeTrue {
					progressingNNCP[nncp.Definition.Name] = condition.Message
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Progressing: %s\n", nncp.Definition.Name, condition.Message))
				} else {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Not-Progressing\n", nncp.Definition.Name))
				}
			}
		}
	}

	Expect(len(nonAvailableNNCP)).To(Equal(0), "There are NonAvailable NodeNetworkConfigurationPolicies")
	Expect(len(degradedNNCP)).To(Equal(0), "There are Degraded NodeNetworkConfigurationPolicies")
	Expect(len(nonAvailableNNCP)).To(Equal(0), "There are Progressing NodeNetworkConfigurationPolicies")
}
