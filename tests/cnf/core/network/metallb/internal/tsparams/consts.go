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
	// DaemonsFile represents FRR default daemon configuration template.
	DaemonsFile = `
	# This file tells the frr package which daemons to start.
    #
    # Sample configurations for these daemons can be found in
    # /usr/share/doc/frr/examples/.
    #
    # ATTENTION:
    #
    # When activating a daemon for the first time, a config file, even if it is
    # empty, has to be present *and* be owned by the user and group "frr", else
    # the daemon will not be started by /etc/init.d/frr. The permissions should
    # be u=rw,g=r,o=.
    # When using "vtysh" such a config file is also needed. It should be owned by
    # group "frrvty" and set to ug=rw,o= though. Check /etc/pam.d/frr, too.
    #
    # The watchfrr, zebra and staticd daemons are always started.
    #
    bgpd=yes
    ospfd=no
    ospf6d=no
    ripd=no
    ripngd=no
    isisd=no
    pimd=no
    ldpd=no
    nhrpd=no
    eigrpd=no
    babeld=no
    sharpd=no
    pbrd=no
    bfdd=yes
    fabricd=no
    vrrpd=no
    pathd=no
    #
    # If this option is set the /etc/init.d/frr script automatically loads
    # the config via "vtysh -b" when the servers are started.
    # Check /etc/pam.d/frr if you intend to use "vtysh"!
    #
    vtysh_enable=yes
    zebra_options="  -A 127.0.0.1 -s 90000000"
    bgpd_options="   -A 127.0.0.1"
    ospfd_options="  -A 127.0.0.1"
    ospf6d_options=" -A ::1"
    ripd_options="   -A 127.0.0.1"
    ripngd_options=" -A ::1"
    isisd_options="  -A 127.0.0.1"
    pimd_options="   -A 127.0.0.1"
    ldpd_options="   -A 127.0.0.1"
    nhrpd_options="  -A 127.0.0.1"
    eigrpd_options=" -A 127.0.0.1"
    babeld_options=" -A 127.0.0.1"
    sharpd_options=" -A 127.0.0.1"
    pbrd_options="   -A 127.0.0.1"
    staticd_options="-A 127.0.0.1"
    bfdd_options="   -A 127.0.0.1"
    fabricd_options="-A 127.0.0.1"
    vrrpd_options="  -A 127.0.0.1"
    pathd_options="  -A 127.0.0.1"
    # configuration profile
    #
    #frr_profile="traditional"
    #frr_profile="datacenter"
    #
    # This is the maximum number of FD's that will be available.
    # Upon startup this is read by the control files and ulimit
    # is called.  Uncomment and use a reasonable value for your
    # setup if you are expecting a large number of peers in
    # say BGP.
    #MAX_FDS=1024
    # The list of daemons to watch is automatically generated by the init script.
    #watchfrr_options=""
    # To make watchfrr create/join the specified netns, use the following option:
    #watchfrr_options="--netns"
    # This only has an effect in /etc/frr/<somename>/daemons, and you need to
    # start FRR with "/usr/lib/frr/frrinit.sh start <somename>".
    # for debugging purposes, you can specify a "wrap" command to start instead
    # of starting the daemon directly, e.g. to use valgrind on ospfd:
    #   ospfd_wrap="/usr/bin/valgrind"
    # or you can use "all_wrap" for all daemons, e.g. to use perf record:
    #   all_wrap="/usr/bin/perf record --call-graph -"
    # the normal daemon command is added to this at the end.
	`
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
