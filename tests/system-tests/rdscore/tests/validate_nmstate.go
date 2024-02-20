package rds_core_system_test

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

var _ = Describe(
	"NMState validation",
	Ordered,
	ContinueOnFailure,
	Label(rdscoreparams.LabelValidateNMState), func() {
		It(fmt.Sprintf("Verifies %s namespace exists", RDSCoreConfig.NMStateOperatorNamespace), func() {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				fmt.Sprintf("Verify namespace %q exists", RDSCoreConfig.NMStateOperatorNamespace))

			_, err := namespace.Pull(APIClient, "openshift-nmstate")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", RDSCoreConfig.NMStateOperatorNamespace))
		})

		It("Verifies NMState instance exists", polarion.ID("67027"), func() {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify NMState instance exists")

			_, err := nmstate.PullNMstate(APIClient, rdscoreparams.NMStateInstanceName)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to pull in NMState instance %q", rdscoreparams.NMStateInstanceName))
		})

		It("Verifies all NodeNetworkConfigurationPolicies are Available", func() {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify NodeNetworkConfigurationPolicies are Available")

			const ConditionTypeTrue = "True"

			nncps, err := nmstate.ListPolicy(APIClient, goclient.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to list NodeNetworkConfigurationPolicies: %v", err))
			Expect(len(nncps)).ToNot(Equal(0), "0 NodeNetworkConfigurationPolicies found")

			nonAvailableNNCP := make(map[string]string)
			progressingNNCP := make(map[string]string)
			degradedNNCP := make(map[string]string)

			for _, nncp := range nncps {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
					fmt.Sprintf("\t Processing %s NodeNetworkConfigurationPolicy", nncp.Definition.Name))

				for _, condition := range nncp.Object.Status.Conditions {
					//nolint:nolintlint
					switch condition.Type { //nolint:exhaustive
					//nolint:goconst
					case "Available":
						if condition.Status != ConditionTypeTrue {
							nonAvailableNNCP[nncp.Definition.Name] = condition.Message
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
								fmt.Sprintf("\t%s NNCP is not Available: %s\n", nncp.Definition.Name, condition.Message))
						} else {
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
								fmt.Sprintf("\t%s NNCP is Available: %s\n", nncp.Definition.Name, condition.Message))
						}
					case "Degraded":
						if condition.Status == ConditionTypeTrue {
							degradedNNCP[nncp.Definition.Name] = condition.Message
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
								fmt.Sprintf("\t%s NNCP is Degraded: %s\n", nncp.Definition.Name, condition.Message))
						} else {
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
								fmt.Sprintf("\t%s NNCP is Not-Degraded\n", nncp.Definition.Name))
						}
					case "Progressing":
						if condition.Status == ConditionTypeTrue {
							progressingNNCP[nncp.Definition.Name] = condition.Message
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
								fmt.Sprintf("\t%s NNCP is Progressing: %s\n", nncp.Definition.Name, condition.Message))
						} else {
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
								fmt.Sprintf("\t%s NNCP is Not-Progressing\n", nncp.Definition.Name))
						}
					}

				}

			}
			Expect(len(nonAvailableNNCP)).To(Equal(0), "There are NonAvailable NodeNetworkConfigurationPolicies")
			Expect(len(degradedNNCP)).To(Equal(0), "There are Degraded NodeNetworkConfigurationPolicies")
			Expect(len(nonAvailableNNCP)).To(Equal(0), "There are Progressing NodeNetworkConfigurationPolicies")

		})
	})
