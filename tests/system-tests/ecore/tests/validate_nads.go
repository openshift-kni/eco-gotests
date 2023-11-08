package ecore_system_test

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore MacVlan Definitions",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateNAD), func() {
		BeforeAll(func() {

			By(fmt.Sprintf("Asserting namespace %q exists", ECoreConfig.NamespacePCC))

			_, err := namespace.Pull(APIClient, ECoreConfig.NamespacePCC)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", ECoreConfig.NamespacePCC))

			By(fmt.Sprintf("Asserting namespace %q exists", ECoreConfig.NamespacePCG))

			_, err = namespace.Pull(APIClient, ECoreConfig.NamespacePCG)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", ECoreConfig.NamespacePCG))

		})

		It("Asserts net-attach-def exist in PCC ns", Label("ecore_validate_nad_pcc_ns"), func() {

			for _, nadName := range ECoreConfig.NADListPCC {
				By(fmt.Sprintf("Asserting %q exists in %q ns", nadName, ECoreConfig.NamespacePCC))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Checking NAD %q in %q ns", nadName, ECoreConfig.NamespacePCC)

				_, err := nad.Pull(APIClient, nadName, ECoreConfig.NamespacePCC)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to find net-attach-def %q", nadName))
			}

		})

		It("Asserts net-attach-def exist in PCG ns", Label("ecore_validate_nad_pcg_ns"), func() {

			for _, nadName := range ECoreConfig.NADListPCG {
				By(fmt.Sprintf("Asserting %q exists in %q ns", nadName, ECoreConfig.NamespacePCG))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Checking NAD %q in %q ns", nadName, ECoreConfig.NamespacePCG)

				_, err := nad.Pull(APIClient, nadName, ECoreConfig.NamespacePCG)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to find net-attach-def %q", nadName))
			}

		})
	})
