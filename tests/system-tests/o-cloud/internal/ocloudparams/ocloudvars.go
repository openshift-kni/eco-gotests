package ocloudparams

import (
	hardwaremanagementv1alpha1 "github.com/openshift-kni/oran-o2ims/api/hardwaremanagement/v1alpha1"
	provisioningv1alpha1 "github.com/openshift-kni/oran-o2ims/api/provisioning/v1alpha1"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsparams"
	"github.com/openshift-kni/k8sreporter"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{systemtestsparams.Label, Label}

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"oran-hwmgr-plugin": "oran-hwmgr-plugin",
		"oran-o2ims":        "oran-o2ims",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &provisioningv1alpha1.ClusterTemplateList{}},
		{Cr: &provisioningv1alpha1.ProvisioningRequestList{}},
		{Cr: &hardwaremanagementv1alpha1.HardwareTemplateList{}},
		{Cr: &hardwaremanagementv1alpha1.NodeList{}},
		{Cr: &hardwaremanagementv1alpha1.NodePoolList{}},
	}

	// TemplateName defines the base name of the referenced ClusterTemplate.
	TemplateName = "sno-ran-du"

	// PolicyTemplateParameters defines the policy template parameters.
	PolicyTemplateParameters = map[string]any{}

	//nolint:lll
	// TemplateVersion1 defines the version of the referenced ClusterTemplate used for the successful SNO provisioning using AI.
	TemplateVersion1 = "v4-18-0-ec3-1"

	//nolint:lll
	// TemplateVersion2 defines the version of the referenced ClusterTemplate used for the failing SNO provisioning using AI.
	TemplateVersion2 = "v4-18-0-ec3-2"

	//nolint:lll
	// TemplateVersion3 defines the version of the referenced ClusterTemplate used for the multicluster provisioning with different templates.
	TemplateVersion3 = "v4-18-0-ec3-3"

	//nolint:lll
	// TemplateVersion4 defines the version of the referenced ClusterTemplate used for the successful SNO provisioning using IBI.
	TemplateVersion4 = "v4-18-0-ec3-4"

	//nolint:lll
	// TemplateVersion5 defines the version of the referenced ClusterTemplate used for the failing SNO provisioning using IBI.
	TemplateVersion5 = "v4-18-0-ec3-5"

	// TemplateVersion6 defines the version of the referenced ClusterTemplate used for the Day 2 operations.
	TemplateVersion6 = "v4-18-0-ec3-6"

	//nolint:lll
	// TemplateVersionSeed defines the version of the referenced ClusterTemplate used for the provisioning of the seed cluster for IBI.
	TemplateVersionSeed = "v4-18-0-ec3-seed-1"

	// NodeClusterName1 is the name of the ORAN Node Cluster.
	NodeClusterName1 = "nodeCluster1"

	// NodeClusterName2 is the name of the ORAN Node Cluster.
	NodeClusterName2 = "nodeCluster2"

	// OCloudSiteID is the ID of the of the ORAN O-Cloud Site.
	OCloudSiteID = "site1"

	// ClusterName1 name of the first cluster.
	ClusterName1 = "sno02"

	// ClusterName2 name of the second cluster.
	ClusterName2 = "sno03"

	// SSHCluster2 is the address to ssh the second cluster.
	SSHCluster2 = "sno03.oran.telcoqe.eng.rdu2.dc.redhat.com:22"

	// HostName2 is the hostname of the second cluster.
	HostName2 = "sno03.oran.telcoqe.eng.rdu2.dc.redhat.com"

	// ClusterInstanceParameters1 is the map with the cluster instance parameters for the first cluster.
	ClusterInstanceParameters1 = map[string]any{
		"clusterName": "sno02",
		"nodes": []map[string]any{
			{
				"hostName": "sno02.oran.telcoqe.eng.rdu2.dc.redhat.com",
			},
		},
	}

	// ClusterInstanceParameters2 is the map with the cluster instance parameters for the second cluster.
	ClusterInstanceParameters2 = map[string]any{
		"clusterName": "sno03",
		"nodes": []map[string]any{
			{
				"hostName": "sno03.oran.telcoqe.eng.rdu2.dc.redhat.com",
				"nodeNetwork": map[string]any{
					"config": map[string]any{
						"interfaces": []map[string]any{
							{
								"name":  "ens3f3",
								"type":  "ethertype",
								"state": "up",
								"ipv6": map[string]any{
									"enabled": "true",
									"address": []map[string]any{
										{
											"ip":            "2620:52:9:1698::6",
											"prefix-length": "64",
										},
									},
									"dhcp":     "false",
									"autoconf": "false",
								},
								"ipv4": map[string]any{
									"enabled": "false",
								},
							},
						},
					},
				},
			},
		},
	}

	// PTPVersionMajorOld old major version of the PTP operator.
	PTPVersionMajorOld uint64 = 4
	// PTPVersionMinorOld old minor version of the PTP operator.
	PTPVersionMinorOld uint64 = 18
	// PTPVersionPatchOld old patch version of the PTP operator.
	PTPVersionPatchOld uint64 = 0
	// PTPVersionPrereleaseOld old prerelease version of the PTP operator.
	PTPVersionPrereleaseOld uint64 = 202501230001

	// PTPVersionMajorNew new major version of the PTP operator.
	PTPVersionMajorNew uint64 = 4
	// PTPVersionMinorNew new minor version of the PTP operator.
	PTPVersionMinorNew uint64 = 18
	// PTPVersionPatchNew new patch version of the PTP operator.
	PTPVersionPatchNew uint64 = 0
	// PTPVersionPrereleaseNew new prerelease version of the PTP operator.
	PTPVersionPrereleaseNew uint64 = 202502250302
)
