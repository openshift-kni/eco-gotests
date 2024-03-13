package ecore_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecorecommon"
)

var _ = Describe(
	"All Suite",
	Ordered,
	ContinueOnFailure,
	Label("validate-ecore-suite"), func() {
		Context("Configured Cluster", Label("clean-cluster"), func() {
			It("Verify kernel modules on control-plane nodes", polarion.ID("67036"),
				Label("validate_kernel_modules", "validate_kernel_modules_control_plane"),
				ecorecommon.ValidateKernelModulesOnControlPlane)

			It("Verify kernle modules on standard nodes", polarion.ID("67034"),
				Label("validate_kernel_modules", "validate_kernel_modules_standard"),
				ecorecommon.ValidateKernelModulesOnStandardNodes)

			It("Verifies NMState Operator is installed", polarion.ID("67027"),
				Label("validate-nmstate"), ecorecommon.VerifyNMStateInstanceExists)

			It("Verifies NMState Operator is installed", polarion.ID("72251"),
				Label("validate-nmstate"), ecorecommon.VerifyNMStatePoliciesAvailable)

			It("Verifies all policies are compliant", polarion.ID("72354"), Label("validate-policies"),
				ecorecommon.ValidateAllPoliciesCompliant)

			ecorecommon.VerifyPersistentStorageSuite()

			ecorecommon.VerifySRIOVSuite()

			ecorecommon.VerifyHardRebootSuite()

			ecorecommon.VerifyGracefulRebootSuite()
		})
	})
