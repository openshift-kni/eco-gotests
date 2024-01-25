package ecore_system_test

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore NMState",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateNmstate), func() {
		It(fmt.Sprintf("Verifies %s namespace exists", ecoreparams.NMStateNS), func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("Verify namespace %q exists", ecoreparams.NMStateNS))

			_, err := namespace.Pull(APIClient, ecoreparams.NMStateNS)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", ecoreparams.NMStateNS))
		})

		It("Verifies NMState instance exists", func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Verify NMState instance exists")

			_, err := nmstate.PullNMstate(APIClient, ecoreparams.NMStateInstanceName)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to pull in NMState instance %q", ecoreparams.NMStateInstanceName))
		})

		It("Verifies all pods are running", func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("Verify all pods in namespace %q are running", ecoreparams.NMStateNS))

			podsRunning, err := pod.WaitForAllPodsInNamespaceRunning(APIClient,
				ecoreparams.NMStateNS, 3*time.Minute, metav1.ListOptions{})
			Expect(podsRunning).To(BeTrue(),
				fmt.Sprintf("Some pods in %q namespace ain't running", ecoreparams.NMStateNS))
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Error checking pods in %s namespace", ecoreparams.NMStateNS))
		})

		It("Verifies all NodeNetworkConfigurationPolicies are Available", func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Verify NodeNetworkConfigurationPolicies are Available")

			const ConditionTypeTrue = "True"

			nncps, err := nmstate.ListPolicy(APIClient, goclient.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to list NodeNetworkConfigurationPolicies: %v", err))
			Expect(len(nncps)).ToNot(Equal(0), "0 NodeNetworkConfigurationPolicies found")

			nonAvailableNNCP := make(map[string]string)
			progressingNNCP := make(map[string]string)
			degradedNNCP := make(map[string]string)

			for _, nncp := range nncps {
				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("\t Processing %s NodeNetworkConfigurationPolicy", nncp.Definition.Name))

				for _, condition := range nncp.Object.Status.Conditions {
					switch condition.Type {
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

		})

	}) // end Describe Ecore NMState
