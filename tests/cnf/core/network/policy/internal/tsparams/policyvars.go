package tsparams

import (
	"time"

	multinetpolicyapiv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/k8sreporter"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(netparam.Labels, LabelSuite)
	// DefaultTimeout represents the default timeout for most of Eventually/PollImmediate functions.
	DefaultTimeout = 300 * time.Second
	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		NetConfig.SriovOperatorNamespace: NetConfig.SriovOperatorNamespace,
		NetConfig.MultusNamesapce:        NetConfig.MultusNamesapce,
		TestNamespaceName:                "other",
	}
	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &mcfgv1.MachineConfigPoolList{}},
		{Cr: &sriovv1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovv1.SriovNetworkList{}},
		{Cr: &sriovv1.SriovNetworkNodeStateList{}},
		{Cr: &sriovv1.SriovOperatorConfigList{}},
		{Cr: &v1.NetworkAttachmentDefinitionList{}},
		{Cr: &multinetpolicyapiv1.MultiNetworkPolicyList{}},
	}

	// MCOWaitTimeout represents timeout for mco operations.
	MCOWaitTimeout = time.Hour
	// MultiNetworkPolicyDSName represents multi-networkPolicy daemonSet name.
	MultiNetworkPolicyDSName = "multus-networkpolicy"
	// WaitTimeout represents timeout for the most ginkgo Eventually functions.
	WaitTimeout = 3 * time.Minute
	// RetryInterval represents retry interval for the most ginkgo Eventually functions.
	RetryInterval = 3 * time.Second
)
