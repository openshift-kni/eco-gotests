package samsungparams

import (
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsparams"
	"github.com/openshift-kni/k8sreporter"
	securityV1 "github.com/openshift/api/security/v1"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{systemtestsparams.Label, Label}

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{"samsung-vcore-test": "samsung-vcore-test"}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}

	// PossibleWorkerNodeRoles list of the possible worker roles.
	PossibleWorkerNodeRoles = []string{"odf", "samsung-cnf", "samsung-pp"}

	// CnfMCSelector the cnf nodes selector value.
	CnfMCSelector = "machineconfiguration.openshift.io/role: samsung-cnf"

	// PpMCSelectorKey the cnf mc selector key.
	PpMCSelectorKey = "machineconfiguration.openshift.io/role"

	// OdfMCSelector the cnf nodes selector value.
	OdfMCSelector = map[string]string{"machineconfiguration.openshift.io/role": "odf"}

	// CnfNodesSelector the cnf nodes selector value.
	CnfNodesSelector = map[string]string{"node-role.kubernetes.io/samsung-cnf": ""}

	// PpNodesSelector the cnf nodes selector value.
	PpNodesSelector = map[string]string{"node-role.kubernetes.io/samsung-pp": ""}

	// OdfNodesSelector the cnf nodes selector value.
	OdfNodesSelector = map[string]string{"node-role.kubernetes.io/odf": ""}

	// NetworkPoliciesNamespaces list of the possible worker roles.
	NetworkPoliciesNamespaces = []string{"amfmme1", "nrf1", "nssf1", "smf1", "upf1"}

	// NetworkPolicyType the policyType value.
	NetworkPolicyType = netv1.PolicyTypeIngress

	// NetworkPolicyMonitoringNamespaceSelectorMatchLabels the namespaceSelector value.
	NetworkPolicyMonitoringNamespaceSelectorMatchLabels = map[string]string{
		"network.openshift.io/policy-group": "monitoring",
	}

	// CnfSccAllowCapabilities cnf scc AllowCapabilites value.
	CnfSccAllowCapabilities = []v1.Capability{"SYS_PTRACE", "SYS_ADMIN", "NET_ADMIN", "NET_RAW", "NET_BIND_SERVICE"}

	// CnfSccGroups cnf scc Groups value.
	CnfSccGroups = []string{"system:authenticated"}

	// CnfSccDropCapabilities cnf scc DropCapabilites value.
	CnfSccDropCapabilities = []v1.Capability{"KILL", "MKNOD"}

	// CnfSccVolumes cnf scc Volumes value.
	CnfSccVolumes = []securityV1.FSType{"configMap", "downwardAPI", "emptyDir",
		"persistentVolumeClaim", "projected", "secret"}
)
