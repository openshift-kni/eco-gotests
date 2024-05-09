package vcorecommon

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/csv"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

// VerifyNMStateNamespaceExists asserts namespace for NMState operator exists.
func VerifyNMStateNamespaceExists(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify namespace %q exists",
		VCoreConfig.NMStateOperatorNamespace)

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := namespace.Pull(APIClient, VCoreConfig.NMStateOperatorNamespace)
			if pullErr != nil {
				glog.V(vcoreparams.VCoreLogLevel).Infof(
					fmt.Sprintf("Failed to pull in namespace %q - %v",
						VCoreConfig.NMStateOperatorNamespace, pullErr))

				return false, pullErr
			}

			return true, nil
		})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", VCoreConfig.NMStateOperatorNamespace))
} // func VerifyNMStateNamespaceExists (ctx SpecContext)

// VerifyNMStateCSVConditionSucceeded assert that NMState operator deployment succeeded.
func VerifyNMStateCSVConditionSucceeded(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify NMState operator deployment succeeded")

	nmstateCSVName, err := csv.GetCurrentCSVNameFromSubscription(APIClient,
		vcoreparams.NMStateOperatorName,
		VCoreConfig.NMStateOperatorNamespace)

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get nmstate %s csv name from the %s namespace",
		vcoreparams.NMStateOperatorName, VCoreConfig.NMStateOperatorNamespace))

	nmstateCSVObj, err := olm.PullClusterServiceVersion(APIClient, nmstateCSVName, VCoreConfig.NMStateOperatorNamespace)

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q csv from the %s namespace",
		nmstateCSVName, VCoreConfig.NMStateOperatorNamespace))

	isSuccessful, err := nmstateCSVObj.IsSuccessful()

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to verify nmstate csv %s in the namespace %s status",
			nmstateCSVName, VCoreConfig.NMStateOperatorNamespace))
	Expect(isSuccessful).To(Equal(true),
		fmt.Sprintf("Failed to deploy nmstate operator; the csv %s in the namespace %s status %v",
			nmstateCSVName, VCoreConfig.NMStateOperatorNamespace, isSuccessful))
} // func VerifyNMStateCSVConditionSucceeded (ctx SpecContext)

// VerifyNMStateInstanceExists assert that NMState instance exists.
func VerifyNMStateInstanceExists(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify NMState instance exists")

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := nmstate.PullNMstate(APIClient, vcoreparams.NMStateInstanceName)
			if pullErr != nil {
				glog.V(vcoreparams.VCoreLogLevel).Infof(
					fmt.Sprintf("Failed to pull in NMState instance %q - %v",
						vcoreparams.NMStateInstanceName, pullErr))

				return false, pullErr
			}

			return true, nil
		})

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull in NMState instance %q", vcoreparams.NMStateInstanceName))
} // func VerifyNMStateInstanceExists (ctx SpecContext)

// VerifyAllNNCPsAreOK assert all available NNCPs are Available, not progressing and not degraded.
func VerifyAllNNCPsAreOK(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify NodeNetworkConfigurationPolicies are Available")

	const ConditionTypeTrue = "True"

	nncps, err := nmstate.ListPolicy(APIClient, goclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to list NodeNetworkConfigurationPolicies: %v", err))
	Expect(len(nncps)).ToNot(Equal(0), "0 NodeNetworkConfigurationPolicies found")

	nonAvailableNNCP := make(map[string]string)
	progressingNNCP := make(map[string]string)
	degradedNNCP := make(map[string]string)

	for _, nncp := range nncps {
		glog.V(vcoreparams.VCoreLogLevel).Infof(
			fmt.Sprintf("\t Processing %s NodeNetworkConfigurationPolicy", nncp.Definition.Name))

		for _, condition := range nncp.Object.Status.Conditions {
			//nolint:nolintlint
			switch condition.Type { //nolint:exhaustive
			//nolint:goconst
			case "Available":
				if condition.Status != ConditionTypeTrue {
					nonAvailableNNCP[nncp.Definition.Name] = condition.Message
					glog.V(vcoreparams.VCoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is not Available: %s\n", nncp.Definition.Name, condition.Message))
				} else {
					glog.V(vcoreparams.VCoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Available: %s\n", nncp.Definition.Name, condition.Message))
				}
			case "Degraded":
				if condition.Status == ConditionTypeTrue {
					degradedNNCP[nncp.Definition.Name] = condition.Message
					glog.V(vcoreparams.VCoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Degraded: %s\n", nncp.Definition.Name, condition.Message))
				} else {
					glog.V(vcoreparams.VCoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Not-Degraded\n", nncp.Definition.Name))
				}
			case "Progressing":
				if condition.Status == ConditionTypeTrue {
					progressingNNCP[nncp.Definition.Name] = condition.Message
					glog.V(vcoreparams.VCoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Progressing: %s\n", nncp.Definition.Name, condition.Message))
				} else {
					glog.V(vcoreparams.VCoreLogLevel).Infof(
						fmt.Sprintf("\t%s NNCP is Not-Progressing\n", nncp.Definition.Name))
				}
			}
		}
	}

	Expect(len(nonAvailableNNCP)).To(Equal(0), "There are NonAvailable NodeNetworkConfigurationPolicies")
	Expect(len(degradedNNCP)).To(Equal(0), "There are Degraded NodeNetworkConfigurationPolicies")
	Expect(len(nonAvailableNNCP)).To(Equal(0), "There are Progressing NodeNetworkConfigurationPolicies")
} // func VerifyNNCP (ctx SpecContext)

// VerifyNMStateSuite container that contains tests for NMState verification.
func VerifyNMStateSuite() {
	Describe(
		"NMState validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", VCoreConfig.NMStateOperatorNamespace),
				Label("nmstate"), VerifyNMStateNamespaceExists)

			It("Verifies NMState operator deployment succeeded",
				Label("nmstate"), reportxml.ID("67027"), VerifyNMStateCSVConditionSucceeded)

			It("Verifies NMState instance exists",
				Label("nmstate"), reportxml.ID("67027"), VerifyNMStateInstanceExists)

			It("Verifies all NodeNetworkConfigurationPolicies are Available",
				Label("nmstate"), reportxml.ID("71846"), VerifyAllNNCPsAreOK)
		})
}
