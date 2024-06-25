package mgmtparams

const (
	// Label represents mgmt label that can be used for test cases selection.
	Label = "mgmt"

	// LCANamespace is the namespace used by the lifecycle-agent.
	LCANamespace = "openshift-lifecycle-agent"

	// LCAWorkloadName is the name used for creating resources needed to backup workload app.
	LCAWorkloadName = "ibu-workload-app"

	// LCAOADPNamespace is the namespace used by the OADP operator.
	LCAOADPNamespace = "openshift-adp"

	// LCAKlusterletName is the name of the backup/restore objects related to the klusterlet.
	LCAKlusterletName = "ibu-klusterlet"

	// LCAKlusterletNamespace is the namespace that contains the klusterlet.
	LCAKlusterletNamespace = "open-cluster-management-agent"

	// MGMTLogLevel custom loglevel for the mgmt testing verbose mode.
	MGMTLogLevel = 50
)
