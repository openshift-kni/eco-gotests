package ecore_system_test

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/mco"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore System-Reserved Memory Validation",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateSysReservedMemory), func() {
		It("Asserts KubeletConfig for control-plane nodes", polarion.ID("67061"), func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** Assert KubeletConfig for control-plane nodes")
			glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling %q KubeletConfig", ECoreConfig.KubeletConfigCPName)

			kubeletConfig, err := mco.PullKubeletConfig(APIClient, ECoreConfig.KubeletConfigCPName)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull in KubeletConfig %q",
				ECoreConfig.KubeletConfigCPName))

			glog.V(ecoreparams.ECoreLogLevel).Infof("Check 'autoSizingReserved' is set to 'true'")
			Expect(*kubeletConfig.Definition.Spec.AutoSizingReserved).To(BeTrue(), "'autoSizingReserved' is 'false'")

		})

		It("Asserts KubeletConfig for standard nodes", polarion.ID("67075"), func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** Assert KubeletConfig for standard nodes")
			glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling %q KubeletConfig", ECoreConfig.KubeletConfigStandardName)

			kubeletConfig, err := mco.PullKubeletConfig(APIClient, ECoreConfig.KubeletConfigStandardName)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull in KubeletConfig %q",
				ECoreConfig.KubeletConfigCPName))

			glog.V(ecoreparams.ECoreLogLevel).Infof("Check 'autoSizingReserved' is set to 'true'")
			Expect(*kubeletConfig.Definition.Spec.AutoSizingReserved).To(BeTrue(), "'autoSizingReserved' is 'false'")

		})
	})
