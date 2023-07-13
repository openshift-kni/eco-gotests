package tsparams

import (
	"time"

	sriovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/k8sreporter"
)

var (
	// TestNamespaceName SR-IOV namespace where all test cases are performed.
	TestNamespaceName = "sriov-tests"
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(netparam.Labels, LabelSuite)
	// OperatorConfigDaemon defaults SR-IOV config daemon daemonset.
	OperatorConfigDaemon = "sriov-network-config-daemon"
	// OperatorDevicePlugin defaults SR-IOV device plugin daemonset.
	OperatorDevicePlugin = "sriov-device-plugin"
	// OperatorWebhook defaults SR-IOV webhook daemonset.
	OperatorWebhook = "operator-webhook"
	// OperatorResourceInjector defaults SR-IOV network resource injector daemonset.
	OperatorResourceInjector = "network-resources-injector"
	// OperatorSriovDaemonsets represents all default SR-IOV operator daemonset names.
	OperatorSriovDaemonsets = []string{OperatorConfigDaemon, OperatorDevicePlugin,
		OperatorWebhook, OperatorResourceInjector}
	// ReporterNamespacesToDump tells to the reporter what namespaces to dump.
	ReporterNamespacesToDump = map[string]string{
		NetConfig.SriovOperatorNamespace: NetConfig.SriovOperatorNamespace,
		TestNamespaceName:                "other",
	}
	// DefaultTimeout represents the default timeout for most of Eventually/PollImmediate functions.
	DefaultTimeout = 300 * time.Second
	// SriovStableTimeout defines the timeout duration for stabilizing SR-IOV after applying
	// or deleting SriovNodePolicies.
	SriovStableTimeout = 35 * time.Minute
	// DefaultStableDuration represents the default stableDuration for most StableFor functions.
	DefaultStableDuration = 10 * time.Second
	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &sriovV1.SriovOperatorConfigList{}},
		{Cr: &sriovV1.SriovNetworkNodeStateList{}},
		{Cr: &sriovV1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovV1.SriovNetworkList{}},
	}
)
