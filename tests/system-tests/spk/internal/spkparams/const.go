package spkparams

const (
	// Label is used to select system tests for SPK operator.
	Label = "spk"
	// LabelSPKIngress is used to select all tests for SPK ingress functionality.
	LabelSPKIngress = "spkingress"
	// LabelSPKIngressTCP is used to select all tests for SPK ingress TCP functionality.
	LabelSPKIngressTCP = "spkingresstcp"
	// LabelSPKIngressUDP is used to select all tests for SPK ingress UDP functionality.
	LabelSPKIngressUDP = "spkingressudp"
	// LabelSPKDnsNat46 is used to select all tests for DNS/NAT46 functionality.
	LabelSPKDnsNat46 = "spkdnsnat46"
	// SPKLogLevel configures logging level for SPK related tests.
	SPKLogLevel = 90

	// ConditionTypeReadyString constant to fix linter warning.
	ConditionTypeReadyString = "Ready"

	// ConstantTrueString constant to fix linter warning.
	ConstantTrueString = "True"
)
