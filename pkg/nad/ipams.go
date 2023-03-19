package nad

// IPAMStatic returns static ipam type.
func IPAMStatic() *IPAM {
	return &IPAM{Type: "static"}
}
