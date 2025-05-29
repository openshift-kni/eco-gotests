package tsparams

import (
	"time"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/k8sreporter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(netparam.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"openshift-performance-addon-operator": "performance",
		NetConfig.MlbOperatorNamespace:         "metallb-system",
		TestNamespaceName:                      "other",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: setUnstructured(metallb.IPAddressPoolList)},
		{Cr: setUnstructured("L2AdvertisementList")},
		{Cr: setUnstructured(metallb.BGPAdvertisementListKind)},
		{Cr: setUnstructured(metallb.BFDProfileList)},
		{Cr: setUnstructured(metallb.BGPPeerListKind)},
		{Cr: setUnstructured(metallb.MetalLBList)},
		{Cr: setUnstructured("FRRConfigurationList")},
		{Cr: &appsv1.DaemonSetList{}, Namespace: &NetConfig.MlbOperatorNamespace},
		{Cr: &appsv1.DeploymentList{}, Namespace: &NetConfig.Frrk8sNamespace},
		{Cr: &nadv1.NetworkAttachmentDefinitionList{}, Namespace: &TestNamespaceName},
		{Cr: &corev1.ConfigMapList{}, Namespace: &TestNamespaceName},
		{Cr: &corev1.ServiceList{}, Namespace: &TestNamespaceName},
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
	// SpeakerLabel represent label of metallb speaker pods.
	SpeakerLabel = "component=speaker"
	// FRRK8sNodeLabel represents the default metalLb FRRK8S node label.
	FRRK8sNodeLabel = "app=frr-k8s"
	// ExternalMacVlanNADName represents default external NetworkAttachmentDefinition name.
	ExternalMacVlanNADName = "external"
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
	// LBipv4Range represents the LoadBalancer IPv4 Address Pool.
	LBipv4Range = []string{"3.3.3.1", "3.3.3.240"}
	// ExtFrrConnectedPool represents custom network prefixes to advertise from external FRR pod.
	ExtFrrConnectedPool = []string{"80.80.80.80/32", "40.40.40.40/32"}
)

func setUnstructured(kind string) *unstructured.UnstructuredList {
	resource := &unstructured.UnstructuredList{}

	gvk := metallb.GetMetalLbIoGVR().GroupVersion().WithKind(kind)

	if kind == "FRRConfigurationList" {
		gvk = metallb.GetFrrConfigurationGVR().GroupVersion().WithKind(kind)
	}

	resource.SetGroupVersionKind(gvk)

	return resource
}
