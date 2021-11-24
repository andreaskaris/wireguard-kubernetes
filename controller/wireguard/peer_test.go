package wireguard

import (
	"fmt"
	"net"
	"testing"
)

func TestPeerList(t *testing.T) {
	peer1 := Peer{
		PeerHostname:  "host1",
		PeerInnerIp:   net.ParseIP("192.168.0.1"),
		PeerOuterIp:   net.ParseIP("10.0.0.1"),
		PeerOuterPort: 10000,
		PeerPublicKey: "pub1",
		PeerPodSubnet: "priv1",
	}
	peer2 := Peer{
		PeerHostname:  "host2",
		PeerInnerIp:   net.ParseIP("192.168.0.1"),
		PeerOuterIp:   net.ParseIP("10.0.0.1"),
		PeerOuterPort: 10000,
		PeerPublicKey: "pub1",
		PeerPodSubnet: "priv1",
	}
	peer3 := Peer{
		PeerHostname:  "host1",
		PeerInnerIp:   net.ParseIP("192.168.0.3"),
		PeerOuterIp:   net.ParseIP("10.0.0.3"),
		PeerOuterPort: 10000,
		PeerPublicKey: "pub1",
		PeerPodSubnet: "priv1",
	}

	pl := NewPeerList()
	err := pl.UpdateOrAdd(&peer1)
	if err != nil {
		t.Fatal(fmt.Sprintf("pl.UpdateOrAdd(peer1): Expected to return nil error, instead got %s", err))
	}
	err = pl.UpdateOrAdd(&peer2)
	if err != nil {
		t.Fatal(fmt.Sprintf("pl.UpdateOrAdd(peer2): Expected to return nil error, instead got %s", err))
	}
	err = pl.UpdateOrAdd(&peer3)
	if err != nil {
		t.Fatal(fmt.Sprintf("pl.UpdateOrAdd(peer3): Expected to return nil error, instead got %s", err))
	}
	err = pl.Delete("host2")
	if err != nil {
		t.Fatal(fmt.Sprintf("pl.Delete(host2): Expected to return nil error, instead got %s", err))
	}
	if len(*pl) != 1 {
		t.Fatal("TestPeerList(): Expected length for PeerList is 1")
	}
	peer, err := pl.Get("host1")
	if err != nil {
		t.Fatal(fmt.Sprintf("pl.Get(host1): Expected to retrieve an entry, got an error instead: %s", err))
	}
	if peer.PeerInnerIp.String() != "192.168.0.3" {
		t.Fatal(fmt.Sprintf("TestPeerList(): Expected peer.PeerInnerIp to be %s, got %s instead", "192.168.0.3", peer.PeerInnerIp.String()))
	}
}
