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
	TotalVfs int `yaml:"total-vfs"`
}
