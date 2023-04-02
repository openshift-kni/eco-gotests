package tsparams

import (
	metalLbOperatorV1Beta1 "github.com/metallb/metallb-operator/api/v1beta1"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/k8sreporter"
	metalLbV1Beta1 "go.universe.tf/metallb/api/v1beta1"
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
		{Cr: &metalLbV1Beta1.IPAddressPoolList{}},
		{Cr: &metalLbV1Beta1.BGPAdvertisementList{}},
		{Cr: &metalLbV1Beta1.L2AdvertisementList{}},
		{Cr: &metalLbV1Beta1.AddressPoolList{}},
		{Cr: &metalLbV1Beta1.BFDProfileList{}},
		{Cr: &metalLbV1Beta1.BGPPeerList{}},
		{Cr: &metalLbOperatorV1Beta1.MetalLBList{}},
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
	// ExternalMacVlanNADName represents default external NetworkAttachmentDefinition name.
	ExternalMacVlanNADName = "external"
)
