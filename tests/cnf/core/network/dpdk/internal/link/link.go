package link

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type link struct {
	Ifindex   int      `json:"ifindex,omitempty"`
	Link      string   `json:"link,omitempty"`
	Ifname    string   `json:"ifname,omitempty"`
	Flags     []string `json:"flags,omitempty"`
	Mtu       int      `json:"mtu,omitempty"`
	Qdisc     string   `json:"qdisc,omitempty"`
	Operstate string   `json:"operstate,omitempty"`
	Linkmode  string   `json:"linkmode,omitempty"`
	Group     string   `json:"group,omitempty"`
	LinkType  string   `json:"link_type,omitempty"`
	Address   string   `json:"address,omitempty"`
	Broadcast string   `json:"broadcast,omitempty"`
	Stats64   struct {
		Rx struct {
			Bytes      int `json:"bytes,omitempty"`
			Packets    int `json:"packets,omitempty"`
			Errors     int `json:"errors,omitempty"`
			Dropped    int `json:"dropped,omitempty"`
			OverErrors int `json:"over_errors,omitempty"`
			Multicast  int `json:"multicast,omitempty"`
		} `json:"rx,omitempty"`
		Tx struct {
			Bytes         int `json:"bytes,omitempty"`
			Packets       int `json:"packets,omitempty"`
			Errors        int `json:"errors,omitempty"`
			Dropped       int `json:"dropped,omitempty"`
			CarrierErrors int `json:"carrier_errors,omitempty"`
			Collisions    int `json:"collisions,omitempty"`
		} `json:"tx,omitempty"`
	} `json:"stats64,omitempty"`
}

type links []link

// NewBuilder returns Link struct.
func NewBuilder(jsonOutput bytes.Buffer) (*link, error) {
	var link links

	err := json.Unmarshal(jsonOutput.Bytes(), &link)
	if err != nil {
		return nil, err
	}

	if len(link) > 1 {
		return nil, fmt.Errorf("failed to process more than 1 link")
	}

	return &link[0], nil
}

// GetRxByte returns number of unicast bytes received on link.
func (l *link) GetRxByte() int {
	return l.Stats64.Rx.Bytes
}
