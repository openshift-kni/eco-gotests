package spk_system_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/spk/internal/spkcommon"
)

var _ = Describe(
	"SPK DNS/NAT46",
	Ordered,
	ContinueOnFailure,
	Label("spk-all-suite"), func() {
		It("Asserts workload reachable via IPv4 address", polarion.ID("64119"),
			Label("spkingresstcp"), spkcommon.AssertIPv4WorkloadURL)

		It("Asserts workload reachable via IPv6 address", polarion.ID("65886"),
			Label("spkingresstcp"), spkcommon.AssertIPv6WorkloadURL)

		It("Asserts DNS resoulution from new deployment", polarion.ID("63171"),
			Label("spkdns46new"), spkcommon.VerifyDNSResolutionFromNewDeploy)

		It("Asserts DNS resolution from existing deployment", polarion.ID("66091"),
			Label("spkdns46existing"), spkcommon.VerifyDNSResolutionFromExistingDeploy)

		It("Asserts DNS Resolution after SPK scale-down and scale-up from existing deployment",
			polarion.ID("72139"), Label("spkscaledown"), spkcommon.VerifyDNSResolutionAfterTMMScaleUpDownExisting)

		It("Asserts DNS Resolution after SPK scale-down and scale-up from new deployment",
			polarion.ID("72140"), Label("spkscaledown"), spkcommon.VerifyDNSResolutionFromNewDeploy)

		It("Asserts DNS Resolution with multiple TMM controllers from existing deployment",
			polarion.ID("72141"), Label("spk-multi-tmm-new"), spkcommon.VerifyDNSResolutionWithMultipleTMMsExisting)

		It("Asserts DNS Resolution with multiple TMM controllers from new deployment",
			polarion.ID("72142"), Label("spk-multi-tmm-existing"), spkcommon.VerifyDNSResolutionFromNewDeploy)

		It("Asserts DNS Resolution after Ingress scale-down and scale-up", Label("spk-ingress-scale-down-up"),
			polarion.ID("72143"), spkcommon.VerifyIngressScaleDownUp)

		It("Asserts DNS Resolution after Ingress scale-down and scale-up from new deployment",
			polarion.ID("72144"), Label("spk-multi-tmm-existing"), spkcommon.VerifyDNSResolutionFromNewDeploy)

	})
