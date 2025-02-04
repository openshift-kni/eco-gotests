package tsparams

import (
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/k8sreporter"
	pluginv1alpha1 "github.com/openshift-kni/oran-hwmgr-plugin/api/hwmgr-plugin/v1alpha1"
	provisioningv1alpha1 "github.com/openshift-kni/oran-o2ims/api/provisioning/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
)

var (
	// Labels is the labels applied to all test cases in the suite.
	Labels = append(ranparam.Labels, LabelSuite)

	// ReporterHubNamespacesToDump tells the reporter which namespaces on the hub to collect pod logs from.
	ReporterHubNamespacesToDump = map[string]string{
		TestName:       "",
		O2IMSNamespace: "",
	}

	// ReporterSpokeNamespacesToDump tells the reporter which namespaces on the spoke to collect pod logs from.
	ReporterSpokeNamespacesToDump = map[string]string{
		TestName: "",
	}

	// ReporterHubCRsToDump is the CRs the reporter should dump on the hub.
	ReporterHubCRsToDump = []k8sreporter.CRData{
		{Cr: &pluginv1alpha1.HardwareManagerList{}, Namespace: ptr.To(HardwareManagerNamespace)},
		{Cr: &provisioningv1alpha1.ProvisioningRequestList{}},
		{Cr: &policiesv1.PolicyList{}},
		{Cr: &siteconfigv1alpha1.ClusterInstanceList{}},
	}

	// ReporterSpokeCRsToDump is the CRs the reporter should dump on the spoke.
	ReporterSpokeCRsToDump = []k8sreporter.CRData{
		{Cr: &corev1.ConfigMapList{}, Namespace: ptr.To(TestName)},
		{Cr: &policiesv1.PolicyList{}},
	}
)

var (
	// HwmgrFailedAuthCondition is the condition to match for when the HardwareManager fails to authenticate with
	// the DTIAS.
	HwmgrFailedAuthCondition = metav1.Condition{
		Type:    string(pluginv1alpha1.ConditionTypes.Validation),
		Reason:  string(pluginv1alpha1.ConditionReasons.Failed),
		Status:  metav1.ConditionFalse,
		Message: "401",
	}

	// PRHardwareProvisionFailedCondition is the ProvisioningRequest condition where hardware provisioning failed.
	PRHardwareProvisionFailedCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.PRconditionTypes.HardwareProvisioned),
		Reason: string(provisioningv1alpha1.CRconditionReasons.Failed),
		Status: metav1.ConditionFalse,
	}
	// PRValidationFailedCondition is the ProvisioningRequest condition where ProvisioningRequest validation failed.
	PRValidationFailedCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.PRconditionTypes.Validated),
		Reason: string(provisioningv1alpha1.CRconditionReasons.Failed),
		Status: metav1.ConditionFalse,
	}
	// PRValidationSucceededCondition is the ProvisioningRequest condition where ProvisioningRequest validation
	// succeeded.
	PRValidationSucceededCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.PRconditionTypes.Validated),
		Reason: string(provisioningv1alpha1.CRconditionReasons.Completed),
		Status: metav1.ConditionTrue,
	}
	// PRNodeConfigFailedCondition is the ProvisioningRequest condition where applying the node configuration
	// failed.
	PRNodeConfigFailedCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.PRconditionTypes.HardwareNodeConfigApplied),
		Reason: string(provisioningv1alpha1.CRconditionReasons.NotApplied),
		Status: metav1.ConditionFalse,
	}
	// PRConfigurationAppliedCondition is the ProvisioningRequest condition where applying day2 configuration
	// succeeds.
	PRConfigurationAppliedCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.PRconditionTypes.ConfigurationApplied),
		Reason: string(provisioningv1alpha1.CRconditionReasons.Completed),
		Status: metav1.ConditionTrue,
	}
	// PRCIProcesssedCondition is the ProvisioningRequest condition where the ClusterInstance has successfully been
	// processed.
	PRCIProcesssedCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.PRconditionTypes.ClusterInstanceProcessed),
		Reason: string(provisioningv1alpha1.CRconditionReasons.Completed),
		Status: metav1.ConditionTrue,
	}

	// CTValidationFailedCondition is the ClusterTemplate condition where the validation failed.
	CTValidationFailedCondition = metav1.Condition{
		Type:   string(provisioningv1alpha1.CTconditionTypes.Validated),
		Reason: string(provisioningv1alpha1.CTconditionReasons.Failed),
		Status: metav1.ConditionFalse,
	}
)
