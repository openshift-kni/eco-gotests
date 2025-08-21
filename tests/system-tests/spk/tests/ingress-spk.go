package spk_system_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/internal/url"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/spk/internal/spkinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/spk/internal/spkparams"
)

var _ = Describe(
	"SPK Ingress",
	Label(spkparams.LabelSPKIngress), func() {
		BeforeEach(func() {
			By("Asserting namespace exists")

			_, err := namespace.Pull(APIClient, SPKConfig.Namespace)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", SPKConfig.Namespace))
		})

		It("Asserts workload reachable via IPv4 address", reportxml.ID("64119"), Label("spkingresstcp"), func() {
			if SPKConfig.IngressTCPIPv4URL == "" {
				Skip("IPv4 URL for SPK backed workload not defined")
			}
			_, _, err := url.Fetch(SPKConfig.IngressTCPIPv4URL, "GET")
			Expect(err).ToNot(HaveOccurred(), "Error reaching IPv4 workload: %s", err)
		})

		It("Asserts workload reachable via IPv6 address", reportxml.ID("65886"), Label("spkingresstcp"), func() {
			if SPKConfig.IngressTCPIPv6URL == "" {
				Skip("IPv6 URL for SPK backed workload not defined")
			}
			_, _, err := url.Fetch(SPKConfig.IngressTCPIPv6URL, "GET")
			Expect(err).ToNot(HaveOccurred(), "Error reaching IPv6 workload: %s", err)
		})
	})
