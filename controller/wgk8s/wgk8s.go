package wgk8s

import (
	"context"
	"log"
	"net"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
	"github.com/andreaskaris/wireguard-kubernetes/controller/wireguard"
)

// Run is responsible for running the wgk8s application. This application sets up the wireguard keys,
// the wireguard namespace and tunnel connections, as well as the wireguard bridge interface.
// The CNI plugin will use this infrastructure and plug in the veth endpoints into the wireguard bridge.
func Run(clientset kubernetes.Interface, localHostname, internalRoutingCidr, wireguardPrivateKey, wireguardPublicKey,
	wireguardNamespace, wireguardInterface, wireguardBridge string) {

	// convert internal routing cidr to network
	// the internal routing cidr is the subnet that is assigned to the tunnel interfaces
	_, internalRoutingNet, err := net.ParseCIDR(internalRoutingCidr)
	if err != nil {
		log.Fatal("Cannot parse internal routing cidr: ", err)
	}

	// todo: at the moment, this is hardcoded to a 16 bit subnet mask
	// this makes mapping of IP addresses easier (simply map the last 2 octets of the node outer IP
	// to the internalRouting network)
	// That means that all nodes currently *must* be part of the same /16 machine network
	if internalRoutingNet.Mask.String() != "ffff0000" {
		log.Fatalf("Invalid mask, must be ffff0000 (16 hex), got: %s", internalRoutingNet.Mask.String())
	}

	// key management
	// create wireguard keys if they do not exist, yet
	if err := wireguard.EnsureWireguardKeys(wireguardPrivateKey, wireguardPublicKey); err != nil {
		log.Fatal(err)
	}
	// read the public key
	pubKey, err := os.ReadFile(wireguardPublicKey)
	localPublicKey := strings.TrimSuffix(string(pubKey), "\n")
	if localPublicKey == "" {
		log.Fatal("Cannot read pubkey:", err)
	}

	// annotate the node which belongs to this process with the public key
	klog.V(5).Info("Updating label of node: ", localHostname, " with public key: ", string(localPublicKey))
	if err := wireguard.AddPublicKeyLabel(clientset, localHostname, string(localPublicKey)); err != nil {
		log.Fatal("Cannot add public key annotation to node:", err)
	}

	// get information about local host
	localNode, err := clientset.CoreV1().Nodes().Get(context.TODO(), localHostname, metav1.GetOptions{})
	if err != nil {
		log.Fatal("Cannot retrieve information about local node: ", err)
	}
	localOuterIp, err := utils.GetNodeMachineNetworkIp(localNode)
	if err != nil {
		log.Fatal(err)
	}
	// retried the tunnel inner IP address of this node (currently 10.64.x.y where x.y are the last 2 octets from this node's node local IP)
	localInnerIp := utils.GetInnerToOuterIp(localOuterIp, *internalRoutingNet)

	// set up the local wireguard tunnel
	if err := wireguard.EnsureNamespace(wireguardNamespace); err != nil {
		log.Fatal(err)
	}

	// set brw0's IP address to the first IP address in the node's PodCIDR
	podCidrs, _ := utils.GetPodCidr(localNode)
	bridgeIp, bridgeIpNetmask, err := utils.GetFirstNetworkAddress(podCidrs["ipv4"])
	if err != nil {
		log.Fatal(err)
	}
	// Create the wgb0 bridge
	if err := wireguard.EnsureBridge(wireguardNamespace, wireguardBridge, bridgeIp, bridgeIpNetmask); err != nil {
		log.Fatal(err)
	}
	// Create the wg0 tunnel
	err = wireguard.InitWireguardTunnel(
		wireguardNamespace,
		wireguardInterface,
		localOuterIp,
		10000,
		localInnerIp,
		wireguardPrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	// monitor nodes
	// Create a list of peers of this node.
	peerList := wireguard.NewPeerList()
	nodesWatcher, _ := clientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{})
	for {
		select {
		case event := <-nodesWatcher.ResultChan():
			// every time a node is added, deleted or modified
			if event.Type == watch.Added || event.Type == watch.Deleted || event.Type == watch.Modified {
				// extract node from event
				node := event.Object.(*corev1.Node)

				// extract node name
				peerHostname := node.Name

				// skip this node event if the event is for the local node
				if localHostname == peerHostname {
					continue
				}

				// extract node IPv4 Cidr
				podCidrs, _ := utils.GetPodCidr(node)
				peerPodSubnet := podCidrs["ipv4"]

				// extract public key node annotation
				nodeAnnotations := node.GetAnnotations()
				peerPublicKey, ok := nodeAnnotations["wireguard.kubernetes.io/publickey"]
				if !ok {
					klog.V(5).Info("Could not get annotation for node, skipping: ", peerHostname)
					continue
				}

				// get the peer's IP address
				peerOuterIp, err := utils.GetNodeMachineNetworkIp(node)
				if err != nil {
					klog.V(1).Info(err.Error())
					continue
				}

				// if this is an add or modify, update the peer list
				if event.Type == watch.Added || event.Type == watch.Modified {
					klog.V(5).Info("Peer node added or updated: ", peerHostname)
					peerInnerIp := utils.GetInnerToOuterIp(peerOuterIp, *internalRoutingNet)
					err = peerList.UpdateOrAdd(&wireguard.Peer{
						PeerHostname:  peerHostname,
						PeerOuterIp:   peerOuterIp,
						PeerInnerIp:   peerInnerIp,
						PeerPublicKey: peerPublicKey,
						PeerOuterPort: 10000,
						PeerPodSubnet: peerPodSubnet,
					})
					// if this is a delete, delete the peer from the peer list
				} else {
					klog.V(5).Info("Peer node deleted: ", peerHostname)
					err = peerList.Delete(peerHostname)
				}
				if err != nil {
					log.Fatal(err)
				}

				// write out the changes in the peer list to the node's wg0 port
				err = wireguard.UpdateWireguardTunnelPeers(
					wireguardNamespace,
					wireguardInterface,
					peerList)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
