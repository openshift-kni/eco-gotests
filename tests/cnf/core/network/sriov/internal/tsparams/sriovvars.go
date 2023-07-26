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
	// DefaultRetryInterval represents the default retry interval for most of Eventually/PollImmediate functions.
	DefaultRetryInterval = 5 * time.Second
	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &sriovV1.SriovOperatorConfigList{}},
		{Cr: &sriovV1.SriovNetworkNodeStateList{}},
		{Cr: &sriovV1.SriovNetworkNodePolicyList{}},
		{Cr: &sriovV1.SriovNetworkList{}},
	}
	// ClientIPv4IPAddress represents the full test client IPv4 address.
	ClientIPv4IPAddress = "192.168.0.1/24"
	// ServerIPv4IPAddress represents the full test server IPv4 address.
	ServerIPv4IPAddress = "192.168.0.2/24"
	// ClientIPv6IPAddress represents the full test client IPv6 address.
	ClientIPv6IPAddress = "2001::1/64"
	// ServerIPv6IPAddress represents the full test server IPv6 address.
	ServerIPv6IPAddress = "2001::2/64"
	// ClientMacAddress represents the test client MacAddress.
	ClientMacAddress = "20:04:0f:f1:88:01"
	// ServerMacAddress represents the test server MacAddress.
	ServerMacAddress = "20:04:0f:f1:88:02"
)
