package nmstate

// DesiredState provides struct for the NMState desired state object containing all NMState configuration.
type DesiredState struct {
	Interfaces []NetworkInterface `yaml:"interfaces"`
}

// NetworkInterface provides struct for the NMState interface state object containing interface information.
type NetworkInterface struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	State    string   `yaml:"state"`
	Ethernet Ethernet `yaml:"ethernet"`
}

// Ethernet provides struct for the NMState Interface Ethernet state object containing interface Ethernet information.
type Ethernet struct {
	Sriov Sriov `yaml:"sr-iov"`
}

// Sriov provides struct for the NMState Interface Ethernet Sriov state object containing
// interface Ethernet Sriov information.
type Sriov struct {
	TotalVfs int  `yaml:"total-vfs"`
	Vfs      []Vf `yaml:"vfs,omitempty"`
}

// Vf provides struct for the NMState SR-IOV VF state object containing SR-IOV VF information.
type Vf struct {
	ID         int    `yaml:"id"`
	MacAddress string `yaml:"mac-address"`
	MaxTxRate  int    `yaml:"max-tx-rate"`
	MinTxRate  int    `yaml:"min-tx-rate"`
	Qos        int    `yaml:"qos"`
	SpoofCheck bool   `yaml:"spoof-check"`
	Trust      bool   `yaml:"trust"`
	VlanID     int    `yaml:"vlan-id"`
}
