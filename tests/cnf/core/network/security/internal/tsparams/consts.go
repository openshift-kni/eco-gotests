package tsparams

const (
	// LabelSuite represents nftables label that can be used for test cases selection.
	LabelSuite = "nftables"
	// LabelNftablesTestCases represents nftables custom firewall label that can be used for test cases selection.
	LabelNftablesTestCases = "nftables-custom-rules"
	// CustomFirewallDelete removes all the rules from the custom table.
	CustomFirewallDelete = `table inet custom_table
          delete table inet custom_table
          table inet custom_table {
          }`
	// CustomFirewallIngressPort8888 creates a custom firewall table blocking incoming tcp port 8888.
	CustomFirewallIngressPort8888 = `table inet custom_table
delete table inet custom_table
table inet custom_table {
	chain custom_chain_INPUT {
		type filter hook input priority 1; policy accept;
		# Drop TCP port 8888 and log
		tcp dport 8888 log prefix "[USERFIREWALL] PACKET DROP: " drop
	}
}`

	// CustomFirewallIngressEgressPort8888 creates a custom firewall table blocking incoming and outgoing tcp port 8888.
	CustomFirewallIngressEgressPort8888 = `table inet custom_table
delete table inet custom_table
table inet custom_table {
	chain custom_chain_INPUT {
		type filter hook input priority 1; policy accept;
		# Drop TCP port 8888 and log
		tcp dport 8888 log prefix "[USERFIREWALL] PACKET DROP: " drop
	}
	chain custom_chain_OUTPUT {
		type filter hook output priority 1; policy accept;
		# Drop TCP port 8888 and log
		tcp dport 8888 log prefix "[USERFIREWALL] PACKET DROP: " drop
}`

	// CustomFirewallIngress8888EgressPort8088 creates a custom firewall table blocking ingress
	// port 8888 and egress port 8088.
	CustomFirewallIngress8888EgressPort8088 = `table inet custom_table
delete table inet custom_table
table inet custom_table {
	chain custom_chain_INPUT {
		type filter hook input priority 1; policy accept;
		# Drop TCP port 8888 and log
		tcp dport 8888 log prefix "[USERFIREWALL] PACKET DROP: " drop
	}
	chain custom_chain_OUTPUT {
		type filter hook output priority 1; policy accept;
		# Drop TCP port 8088 and log
		tcp dport 8088 log prefix "[USERFIREWALL] PACKET DROP: " drop
}`
)
