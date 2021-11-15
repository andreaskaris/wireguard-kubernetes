package wireguard

import "net"

/*
wg genkey | tee /etc/wireguard/privatekey | wg pubkey | sudo tee /etc/wireguard/publickey
ip link add wg0 type wireguard
ip a a dev wg0 10.0.0.1/24
wg set wg0 private-key /etc/wireguard/privatekey
ip link set dev wg0 up
wg set wg0 listen-port 50000 peer BH7uivUtp57tZjT+tDPg2khgM5Mu6Ecm/6WQy4Fw2Ew= allowed-ips 10.0.0.2/32 endpoint 172.18.0.5:39332
*/
type Peer struct {
	publicKey  string
	privateKey string
	allowedIPs []*net.IP
	endpoint   []*net.UDPAddr
}
