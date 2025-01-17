package tsparams

const (
	// LabelSuite represents metallb label that can be used for test cases selection.
	LabelSuite = "metallb"
	// LabelBFDTestCases represents bfd label that can be used for test cases selection.
	LabelBFDTestCases = "bfd"
	// LabelBGPTestCases represents bgp label that can be used for test cases selection.
	LabelBGPTestCases = "bgp"
	// LabelDynamicRemoteASTestCases represents dynamic-remote-as label that can be used for test cases selection.
	LabelDynamicRemoteASTestCases = "dynamic-remote-as"
	// LabelLayer2TestCases represents layer2 label that can be used for test cases selection.
	LabelLayer2TestCases = "layer2"
	// LabelFRRTestCases represents frrk8 label that can be used for test cases selection.
	LabelFRRTestCases = "frrk8s"
	// BGPPassword var is used to set password for BGP session between FRR speakers.
	BGPPassword = "bgp-test"
	// LabelGRTestCases represents graceful restart label that can be used for test cases selection.
	LabelGRTestCases = "gracefulrestart"
	// MlbAddressListError an error message when the ECO_CNF_CORE_NET_MLB_ADDR_LIST is incorrect.
	MlbAddressListError = "An unexpected error occurred while " +
		"determining the IP addresses from the ECO_CNF_CORE_NET_MLB_ADDR_LIST environment variable."
	// FRRBaseConfig represents FRR daemon minimal configuration.
	FRRBaseConfig = `!
frr defaults traditional
hostname frr-pod
log file /tmp/frr.log
log timestamp precision 3
!
debug zebra nht
debug bgp neighbor-events
!
bfd
!
`
	// FRRDefaultBGPPreConfig represents FRR daemon BGP minimal config.
	FRRDefaultBGPPreConfig = ` bgp router-id 10.10.10.11
  no bgp ebgp-requires-policy
  no bgp default ipv4-unicast
  no bgp network import-check
`
	// FRRDefaultConfigMapName represents default FRR configMap name.
	FRRDefaultConfigMapName = "frr-config"
	// LocalBGPASN represents local BGP AS number.
	LocalBGPASN = 64500
	// RemoteBGPASN represents remote BGP AS number.
	RemoteBGPASN = 64501
)
