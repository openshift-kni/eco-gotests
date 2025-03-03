package tsparams

import (
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/k8sreporter"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(netparam.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"openshift-performance-addon-operator": "performance",
		"metallb-system":                       "metallb-system",
		"metallb-test":                         "other",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: setUnstructured(metallb.IPAddressPoolList)},
		{Cr: setUnstructured(metallb.BGPAdvertisementListKind)},
		{Cr: setUnstructured(metallb.IPAddressPoolList)},
		{Cr: setUnstructured(metallb.BFDProfileList)},
		{Cr: setUnstructured(metallb.BGPPeerListKind)},
		{Cr: setUnstructured(metallb.MetalLBList)},
	}
	// TestNamespaceName metalLb namespace where all test cases are performed.
	TestNamespaceName = "metallb-tests"
	// OperatorControllerManager defaults metalLb daemonset controller name.
	OperatorControllerManager = "metallb-operator-controller-manager"
	// OperatorWebhook defaults metalLb webhook deployment name.
	OperatorWebhook = "metallb-operator-webhook-server"
	// MetalLbIo default metalLb.io resource name.
	MetalLbIo = "metallb"
	// MetalLbDsName default metalLb speaker daemonset names.
	MetalLbDsName = "speaker"
	// FrrDsName default metalLb frr daemonset name.
	FrrDsName = "frr-k8s"
	// FRRK8sDefaultLabel represents the default metalLb FRRK8S pod label.
	FRRK8sDefaultLabel = "component=frr-k8s"
	// FRRK8sNodeLabel represents the default metalLb FRRK8S node label.
	FRRK8sNodeLabel = "app=frr-k8s"
	// HubMacVlanNADName represents default external NetworkAttachmentDefinition name.
	HubMacVlanNADName = "nad-hub"
	// HubMacVlanNADSecIntName represents a NetworkAttachmentDefinition that includes the master interface.
	HubMacVlanNADSecIntName = "nad-hub-sec-int"
	// SleepCMD represents shel sleep command.
	SleepCMD = []string{"/bin/bash", "-c", "sleep INF"}
	// FRRContainerName represents default FRR's container name.
	FRRContainerName = "frr"
	// FRRSecondContainerName represents second FRR's container name.
	FRRSecondContainerName = "frr2"
	// DefaultTimeout represents the default timeout for most of Eventually/PollImmediate functions.
	DefaultTimeout = 180 * time.Second
	// DefaultRetryInterval represents the default retry interval for most of Eventually/PollImmediate functions.
	DefaultRetryInterval = 10 * time.Second
	// MetalLbSpeakerLabel represents test node label which allows to run metalLb speakers on specific nodes.
	MetalLbSpeakerLabel = map[string]string{"metal": "test"}
	// PrometheusMonitoringLabel represents the label which tells prometheus to monitor a given object.
	PrometheusMonitoringLabel = "openshift.io/cluster-monitoring"
	// PrometheusMonitoringPodLabel represents monitoring pod label selector.
	PrometheusMonitoringPodLabel = "app.kubernetes.io/name=prometheus"
	// EBGPProtocol represents external bgp protocol name.
	EBGPProtocol = "eBGP"
	// IBPGPProtocol represents internal bgp protocol name.
	IBPGPProtocol = "iBGP"
	// TestLabel represents node label for testing.
	TestLabel = map[string]string{"test": "label"}
	// MetalLbBgpMetrics represents the list of expected metallb metrics.
	MetalLbBgpMetrics = []string{"frrk8s_bgp_keepalives_sent", "frrk8s_bgp_keepalives_received",
		"frrk8s_bgp_notifications_sent", "frrk8s_bgp_opens_received", "frrk8s_bgp_opens_sent",
		"frrk8s_bgp_route_refresh_sent", "frrk8s_bgp_session_up", "frrk8s_bgp_total_received", "frrk8s_bgp_total_sent",
		"frrk8s_bgp_updates_total", "frrk8s_bgp_updates_total_received", "frrk8s_bgp_announced_prefixes_total",
		"frrk8s_bgp_received_prefixes_total",
	}
)

func setUnstructured(kind string) *unstructured.UnstructuredList {
	resource := &unstructured.UnstructuredList{}

	gvk := schema.GroupVersionKind{
		Group:   metallb.APIGroup,
		Version: metallb.APIVersion,
		Kind:    kind,
	}

	resource.SetGroupVersionKind(gvk)

	return resource
}
