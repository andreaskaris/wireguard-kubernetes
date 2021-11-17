package wireguard

import (
	"fmt"
	"net"
)

const (
	minPort = 10000
	maxPort = 20000
)

type Peer struct {
	LocalHostname      string
	PeerHostname       string
	LocalInnerIp       net.IP // determine automatically, compare hostnames
	PeerInnerIp        net.IP
	LocalOuterIp       net.IP
	PeerOuterIp        net.IP
	PeerOuterPort      int
	LocalOuterPort     int // determin automatically from range
	PeerPublicKey      string
	LocalPrivateKey    string
	LocalInterfaceName string // wg + localOuterPort
}

type PeerList map[string]*Peer

func NewPeerList() *PeerList {
	pl := make(PeerList)
	return &pl
}

func (pl *PeerList) Get(peerName string) *Peer {
	return (*pl)[peerName]
}

// Delete deletes the pod from the PeerList.
func (pl *PeerList) Delete(hostname string) error {
	delete(*pl, hostname)
	return nil
}

// setKeys sets a Peer entry's keys.
func (pl *PeerList) UpdateOrAdd(p *Peer) error {
	if p.LocalHostname == "" {
		return fmt.Errorf("Must provide a valid LocalHostname")
	}
	(*pl)[p.LocalHostname] = p

	return nil
}
