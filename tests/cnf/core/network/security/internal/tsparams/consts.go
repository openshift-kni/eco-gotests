package tsparams

const (
	// LabelSuite represents nftables label that can be used for test cases selection.
	LabelSuite = "nftables"
	// LabelNftablesTestCases represents nftables custom firewall label that can be used for test cases selection.
	LabelNftablesTestCases = "nftables-custom-rules"
	// CustomFirewallDelete removes all the rules from the custom table.
	// CustomFirewallDelete = `
	//      table inet custom_table
	//      delete table inet custom_table
	//      table inet custom_table {
	//      }`
	CustomFirewallDelete = `data:;base64, ` +
		`H4sIAAAAAAAC/ypJTMpJVcjMSy1RSC4tLsnPjQeLcKWk5qSWpCrgksYhrlDNVcsFCAAA//9SII3uUwAAAA==`
	// CustomFirewallIngressPort8888 adds an input rule blocking TCP port 8888.
	// chain custom_chain_INPUT {
	//    type filter hook input priority 1; policy accept;
	//     # Drop TCP port 8888 and log
	//      tcp dport 8888 log prefix "[USERFIREWALL] PACKET DROP: " drop
	CustomFirewallIngressPort8888 = `data:;base64,` +
		`H4sIAAAAAAAC/3TMwUoDMRDG8XPyFB/1Cbwt9lTaFYpFl3WLB5ESk2k7GDNDnIKL` +
		`9N2lBfG0x+//g8/CeyZwIUM8fZl87q7FJ8pkhCme6PjxLh4Dl796Hbv1Y7cdLuZs` +
		`VMKes1HFUeQDXPRk0MpS2UbczqGSOY4IMZLa3Dt3g1UVxbDsoFINTdM0CCUhy+Fy` +
		`GRXpH7IcoJX2/I3Z6/a57e/Xffuy2Gze0C2WD+2AVf/U3WGGVEW9O/uz/w0AAP//` +
		`kU0CJQUBAAA=`
)
