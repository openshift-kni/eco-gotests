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
		Describe("On clean deployment", Label("spk-vanila-deployment"), func() {
			It("Setups workload for SPK Ingress tests", Label("spk-setup-ingress-workload"),
				spkcommon.SetupSPKBackendWorkload)

			It("Asserts workload reachable via IPv4 address", polarion.ID("64119"),
				Label("spkingresstcp"), spkcommon.AssertIPv4WorkloadURL)

			It("Asserts workload reachable via IPv6 address", polarion.ID("65886"),
				Label("spkingresstcp"), spkcommon.AssertIPv6WorkloadURL)

			It("Asserts workload reachable via IPv4 address after application recreation", polarion.ID("72437"),
				Label("spkingresstcp-app-recreate"), MustPassRepeatedly(3),
				spkcommon.AssertIPv4WorkloadURLAfterAppRecreated)

			It("Asserts workload reachable via IPv6 address after application recreation", polarion.ID("72439"),
				Label("spkingresstcp-app-recreate"), MustPassRepeatedly(3),
				spkcommon.AssertIPv6WorkloadURLAfterAppRecreated)

			It("Asserts DNS resoulution from new deployment", polarion.ID("63171"),
				Label("spkdns46new"), spkcommon.VerifyDNSResolutionFromNewDeploy)

			It("Asserts DNS resolution from existing deployment", polarion.ID("66091"),
				Label("spkdns46existing"), spkcommon.VerifyDNSResolutionFromExistingDeploy)

			It("Asserts DNS Resolution after SPK scale-down and scale-up from existing deployment",
				polarion.ID("72139"), Label("spkscaledown"), spkcommon.VerifyDNSResolutionAfterTMMScaleUpDownExisting)

			It("Asserts DNS Resolution after SPK scale-down and scale-up from new deployment",
				polarion.ID("72140"), Label("spkscaledown"), spkcommon.VerifyDNSResolutionFromNewDeploy)

			It("Asserts DNS Resolution after SPK TMM pod(s) are deleted from existing deployment",
				polarion.ID("72644"), Label("spk-tmm-delete"), spkcommon.VerifyDNSResolutionAfterTMMPodIsDeletedExistingDeploy)

			It("Asserts DNS Resolution after SPK TMM pod(s) are deleted from new deployment",
				polarion.ID("72645"), Label("spk-tmm-delete"), spkcommon.VerifyDNSResolutionAfterTMMPodIsDeletedNewDeploy)

			It("Asserts DNS Resolution with multiple TMM controllers from existing deployment",
				polarion.ID("72141"), Label("spk-multi-tmm-new"), spkcommon.VerifyDNSResolutionWithMultipleTMMsExisting)

			It("Asserts DNS Resolution with multiple TMM controllers from new deployment",
				polarion.ID("72142"), Label("spk-multi-tmm-existing"), spkcommon.VerifyDNSResolutionFromNewDeploy)

			It("Asserts DNS Resolution after Ingress scale-down and scale-up", Label("spk-ingress-scale-down-up"),
				polarion.ID("72143"), spkcommon.VerifyIngressScaleDownUp)

			It("Asserts DNS Resolution after Ingress scale-down and scale-up from new deployment",
				polarion.ID("72144"), Label("spk-multi-tmm-existing"), spkcommon.VerifyDNSResolutionFromNewDeploy)

			It("Assert DNS resolution after Ingress pods were deleted from existing deployment", polarion.ID("72280"),
				Label("spk-ingress-delete-existing"), spkcommon.VerifyDNSResolutionAfterIngressPodIsDeleteExistinDeploy)

			It("Assert DNS resolution after Ingress pods were deleted from new deployment", polarion.ID("72283"),
				Label("spk-ingress-delete-existing"), spkcommon.VerifyDNSResolutionAfterIngressPodIsDeleteNewDeploy)

			It("Assert workload is reachable over IPv4 SPK ingress after pod was deleted", polarion.ID("72278"),
				Label("spk-ingress-delete-ipv4"),
				spkcommon.AssertIPv4WorkloadURLAfterIngressPodDeleted)

			It("Assert workload is reachable over IPv6 SPK ingress after pod was deleted", polarion.ID("72279"),
				Label("spk-ingress-delete"),
				spkcommon.AssertIPv6WorkloadURLAfterIngressPodDeleted)
		})

		Describe("Hard reboot", Label("spk-hard-reboot"), func() {
			It("Setups workload for SPK Ingress tests", Label("spk-setup-ingress-workload"),
				spkcommon.SetupSPKBackendWorkload)

			spkcommon.VerifyHardRebootSuite()

			// https://issues.redhat.com/browse/OCPBUGS-30170
			It("Removes stuck SPK pods",
				Label("spk-post-hard-reboot", "spk-cleanup-stuck-pods"), spkcommon.CleanupStuckContainerPods)

			It("Asserts workload reachable via IPv4 address", polarion.ID("72193"),
				Label("spk-post-hard-reboot", "spkingresstcp"), spkcommon.AssertIPv4WorkloadURL)

			It("Asserts workload reachable via IPv6 address", polarion.ID("72194"),
				Label("spk-post-hard-reboot", "spkingresstcp"), spkcommon.AssertIPv6WorkloadURL)

			It("Asserts DNS resoulution from new deployment", polarion.ID("72196"),
				Label("spk-post-hard-reboot", "spkdns46new"), spkcommon.VerifyDNSResolutionFromNewDeploy)

			It("Asserts DNS resolution from existing deployment", polarion.ID("72197"),
				Label("spk-post-hard-reboot", "spkdns46existing"), spkcommon.VerifyDNSResolutionFromExistingDeploy)

			It("Asserts workload reachable via IPv4 address after hard reboot and application recreation",
				Label("spkingresstcp-app-recreate"), MustPassRepeatedly(3), polarion.ID("72438"),
				spkcommon.AssertIPv4WorkloadURLAfterAppRecreated)

			It("Asserts workload reachable via IPv6 address after hard reboot and application recreation",
				Label("spkingresstcp-app-recreate"), MustPassRepeatedly(3), polarion.ID("72440"),
				spkcommon.AssertIPv6WorkloadURLAfterAppRecreated)

			It("Asserts DNS Resolution after soft reboot and SPK TMM pod(s) are deleted from existing deployment",
				polarion.ID("72648"), Label("spk-tmm-delete"), spkcommon.VerifyDNSResolutionAfterTMMPodIsDeletedExistingDeploy)

			It("Asserts DNS Resolution after soft reboot and SPK TMM pod(s) are deleted from new deployment",
				polarion.ID("72649"), Label("spk-tmm-delete"), spkcommon.VerifyDNSResolutionAfterTMMPodIsDeletedNewDeploy)
		})

		Describe("Soft Reboot", Label("spk-soft-reboot"), func() {
			spkcommon.VerifyGracefulRebootSuite()

			It("Asserts workload reachable via IPv4 address", polarion.ID("72198"),
				Label("spk-post-soft-reboot", "spkingresstcp"), spkcommon.AssertIPv4WorkloadURL)

			It("Asserts workload reachable via IPv6 address", polarion.ID("72199"),
				Label("spk-post-soft-reboot", "spkingresstcp"), spkcommon.AssertIPv6WorkloadURL)

			It("Asserts DNS resoulution from new deployment", polarion.ID("72200"),
				Label("spk-post-soft-reboot", "spkdns46new"), spkcommon.VerifyDNSResolutionFromNewDeploy)

			It("Asserts DNS resolution from existing deployment", polarion.ID("72201"),
				Label("spk-post-soft-reboot", "spkdns46existing"), spkcommon.VerifyDNSResolutionFromExistingDeploy)

			It("Asserts workload reachable via IPv4 address after soft reboot and application recreation",
				Label("spkingresstcp-app-recreate"), MustPassRepeatedly(3), polarion.ID("72441"),
				spkcommon.AssertIPv4WorkloadURLAfterAppRecreated)

			It("Asserts workload reachable via IPv6 address after soft reboot and application recreation",
				Label("spkingresstcp-app-recreate"), MustPassRepeatedly(3), polarion.ID("72442"),
				spkcommon.AssertIPv6WorkloadURLAfterAppRecreated)

			It("Asserts DNS Resolution after soft reboot and SPK TMM pod(s) are deleted from existing deployment",
				polarion.ID("72646"), Label("spk-tmm-delete"), spkcommon.VerifyDNSResolutionAfterTMMPodIsDeletedExistingDeploy)

			It("Asserts DNS Resolution after soft reboot and SPK TMM pod(s) are deleted from new deployment",
				polarion.ID("72647"), Label("spk-tmm-delete"), spkcommon.VerifyDNSResolutionAfterTMMPodIsDeletedNewDeploy)
		})
	})
