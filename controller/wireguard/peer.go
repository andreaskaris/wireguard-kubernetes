package wireguard

import (
	"fmt"
	"net"
)

// Peer is a structure representing a wireguard peer (the node on the other side of the tunnel).
type Peer struct {
	PeerHostname  string
	PeerInnerIp   net.IP
	PeerOuterIp   net.IP
	PeerOuterPort int
	PeerPublicKey string
	PeerPodSubnet string
}

// PeerList is a list of peers.
type PeerList map[string]*Peer

// NewPeerList returns a pointer to a new peer list.
func NewPeerList() *PeerList {
	pl := make(PeerList)
	return &pl
}

// Get retrieves a peer entry.
func (pl *PeerList) Get(peerName string) (*Peer, error) {
	entry, ok := (*pl)[peerName]
	if !ok {
		return nil, fmt.Errorf("No such entry: %s", peerName)
	} else {
		return entry, nil
	}
}

// Delete deletes the pod from the PeerList.
func (pl *PeerList) Delete(hostname string) error {
	delete(*pl, hostname)
	return nil
}

// UpdateOrAdd replaces the peer entry with a new peer entry.
func (pl *PeerList) UpdateOrAdd(p *Peer) error {
	(*pl)[p.PeerHostname] = p

	return nil
}
