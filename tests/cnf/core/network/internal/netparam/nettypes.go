package netparam

type (
	// BFDDescription defines struct for BFD status and respective peer.
	BFDDescription struct {
		BFDStatus string `json:"status"`
		BFDPeer   string `json:"peer"`
	}
)
