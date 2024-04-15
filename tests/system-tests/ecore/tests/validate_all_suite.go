package ecore_system_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecorecommon"
)

var _ = Describe(
	"All Suite",
	Ordered,
	ContinueOnFailure,
	Label("validate-ecore-suite"), func() {
		Context("Configured Cluster", Label("clean-cluster"), func() {
			It("Verify MACVLAN", Label("validate-new-macvlan-different-nodes"), reportxml.ID("72566"),
				ecorecommon.VerifyMacVlanOnDifferentNodes)

			It("Verify MACVLAN", Label("validate-new-macvlan-same-node"), reportxml.ID("72567"),
				ecorecommon.VerifyMacVlanOnSameNode)

			It("Verify kernel modules on control-plane nodes", reportxml.ID("67036"),
				Label("validate_kernel_modules", "validate_kernel_modules_control_plane"),
				ecorecommon.ValidateKernelModulesOnControlPlane)

			It("Verify kernle modules on standard nodes", reportxml.ID("67034"),
				Label("validate_kernel_modules", "validate_kernel_modules_standard"),
				ecorecommon.ValidateKernelModulesOnStandardNodes)

			It("Verifies NMState Operator is installed", reportxml.ID("67027"),
				Label("validate-nmstate"), ecorecommon.VerifyNMStateInstanceExists)

			It("Verifies NMState Operator is installed", reportxml.ID("72251"),
				Label("validate-nmstate"), ecorecommon.VerifyNMStatePoliciesAvailable)

			It("Verifies all policies are compliant", reportxml.ID("72354"), Label("validate-policies"),
				ecorecommon.ValidateAllPoliciesCompliant)

			ecorecommon.VerifyPersistentStorageSuite()

			ecorecommon.VerifySRIOVSuite()

			ecorecommon.VerifyHardRebootSuite()

			ecorecommon.VerifyGracefulRebootSuite()
		})
	})
