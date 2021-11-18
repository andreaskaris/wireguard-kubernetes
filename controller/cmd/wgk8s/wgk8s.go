package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
	"github.com/andreaskaris/wireguard-kubernetes/controller/wireguard"
)

var kubeconfig = flag.String("kubeconfig", "", "Location of kubeconfig file")
var wireguardPrivateKey = flag.String("wg-private-key", "/etc/wireguard/private", "Location of the wireguard private key")
var wireguardPublicKey = flag.String("wg-public-key", "/etc/wireguard/public", "Location of the wireguard public key")
var wireguardNamespace = flag.String("wg-namespace", "wireguard-kubernetes", "Name of the wireguard-kubernetes namespace")
var wireguardInterface = flag.String("wg-interface", "wg0", "Name of the interface inside the wireguard-kubernetes namespace")
var hostname = flag.String("hostname", func() string { s, _ := os.Hostname(); return s }(), "Hostname of this system")
var internalRoutingCidr = flag.String("internal-routing-cidr", "100.64.0.0/16", "Internal routing network used for the wireguard tunnels")

// var sockaddr = flag.String("sockaddr", "/var/run/wgk8s/wgk8s.sock", "Location wgk8s socket (for CNI communication)")

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.Parse()

	// convert internal routing cidr to network
	_, internalRoutingNet, err := net.ParseCIDR(*internalRoutingCidr)
	if err != nil {
		log.Fatal("Cannot parse internal routing cidr: ", err)
	}
	if internalRoutingNet.Mask.String() != "ffff0000" {
		log.Fatalf("Invalid mask, must be ffff0000 (16 hex), got: %s", internalRoutingNet.Mask.String())
	}

	// set up kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// read public key
	pubKey, err := os.ReadFile(*wireguardPublicKey)
	localPublicKey := strings.TrimSuffix(string(pubKey), "\n")
	if localPublicKey == "" {
		log.Fatal("Cannot read pubkey:", err)
	}

	// annotate the node which belongs to this process with the public key
	klog.V(5).Info("Updating label of node: ", *hostname, " with public key: ", string(localPublicKey))
	if err := wireguard.AddPublicKeyLabel(clientset, *hostname, string(localPublicKey)); err != nil {
		log.Fatal("Cannot add public key annotation to node:", err)
	}

	// get information about local node
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), *hostname, metav1.GetOptions{})
	if err != nil {
		log.Fatal("Cannot retrieve information about local node: ", err)
	}
	localHostname := *hostname
	localOuterIp, err := utils.GetNodeInternalIp(node)
	if err != nil {
		log.Fatal(err)
	}
	localInnerIp := utils.GetInnerToOuterIp(localOuterIp, *internalRoutingNet)

	// set up minimum infrastructure
	if err := wireguard.EnsureWireguardKeys(*wireguardPrivateKey, *wireguardPublicKey); err != nil {
		log.Fatal(err)
	}
	if err := wireguard.EnsureNamespace(*wireguardNamespace); err != nil {
		log.Fatal(err)
	}
	err = wireguard.InitWireguardTunnel(
		*wireguardNamespace,
		*wireguardInterface,
		localOuterIp,
		10000,
		localInnerIp,
		*wireguardPrivateKey)
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
			if event.Type == watch.Added || event.Type == watch.Deleted || event.Type == watch.Modified {
				node := event.Object.(*corev1.Node)
				peerHostname := node.Name
				if localHostname == peerHostname {
					continue
				}
				nodeAnnotations := node.GetAnnotations()
				peerPublicKey, ok := nodeAnnotations["wireguard.kubernetes.io/publickey"]
				if !ok {
					klog.V(5).Info("Could not get annotation for node, skipping: ", peerHostname)
					continue
				}

				peerOuterIp, err := utils.GetNodeInternalIp(node)
				if err != nil {
					klog.V(1).Info(err.Error())
					continue
				}

				if event.Type == watch.Added || event.Type == watch.Modified {
					klog.V(5).Info("Peer node added or updated: ", peerHostname)
					peerInnerIp := utils.GetInnerToOuterIp(peerOuterIp, *internalRoutingNet)
					err = peerList.UpdateOrAdd(&wireguard.Peer{
						LocalHostname:      localHostname,
						PeerHostname:       peerHostname,
						LocalOuterIp:       localOuterIp,
						LocalInnerIp:       localInnerIp,
						PeerOuterIp:        peerOuterIp,
						PeerInnerIp:        peerInnerIp,
						PeerPublicKey:      peerPublicKey,
						LocalPrivateKey:    *wireguardPrivateKey,
						LocalOuterPort:     10000,
						PeerOuterPort:      10000,
						LocalInterfaceName: *wireguardInterface,
					})
				} else {
					klog.V(5).Info("Peer node deleted: ", peerHostname)
					err = peerList.Delete(peerHostname)
				}
				if err != nil {
					log.Fatal(err)
				}

				err = wireguard.UpdateWireguardTunnelPeers(
					*wireguardNamespace,
					*wireguardInterface,
					peerList)

				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
