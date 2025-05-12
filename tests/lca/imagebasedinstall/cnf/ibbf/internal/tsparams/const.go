package tsparams

const (
	// LabelSuite represents upgrade label that can be used for test cases selection.
	LabelSuite = "ibbf"
	// LabelEndToEndUpgrade represents e2e label that can be used for test cases selection.
	LabelEndToEndUpgrade = "e2e"

	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName = "clusters"

	// IBBFTestPath is the name of the path to IBBF test siteconfig
	IBBFTestPath = "ibbf_test"

	// RHACMNamespace is the name of the ACM namespace
	RHACMNamespace = "rhacm"

	// SpokeNamespace is the namespace of the spokecluster
	SpokeNamespace = "helix54"

	// ReinstallGenerationLabel is the name of the clusterinstance generation to trigger IBBF
	ReinstallGenerationLabel = "reinstallV1"
)
