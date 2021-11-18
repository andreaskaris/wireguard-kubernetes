package main

import (
	"context"
	"flag"
	"log"
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
var hostname = flag.String("hostname", func() string { s, _ := os.Hostname(); return s }(), "Hostname of this system")

// var sockaddr = flag.String("sockaddr", "/var/run/wgk8s/wgk8s.sock", "Location wgk8s socket (for CNI communication)")

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.Parse()

	// set up kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// set up minimum infrastructure
	if err := utils.EnsureWireguardKeys(*wireguardPrivateKey, *wireguardPublicKey); err != nil {
		log.Fatal(err)
	}
	if err := utils.EnsureNamespace(*wireguardNamespace); err != nil {
		log.Fatal(err)
	}

	localPrivateKey := "/etc/wireguard/private"

	pubKey, err := os.ReadFile(*wireguardPublicKey)
	localPublicKey := strings.TrimSuffix(string(pubKey), "\n")
	if localPublicKey == "" {
		log.Fatal("Cannot read pubkey:", err)
	}

	// annotate the node which belongs to this process with the public key
	klog.V(5).Info("Updating label of node: ", *hostname, " with public key: ", string(localPublicKey))
	if err := utils.AddPublicKeyLabel(clientset, *hostname, string(localPublicKey)); err != nil {
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

	// Create a list of peers of this node.
	peerList := wireguard.NewPeerList()

	// monitor nodes
	nodesWatcher, _ := clientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{})

	// add a watcher for all nodes
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
					err = peerList.UpdateOrAdd(&wireguard.Peer{
						LocalHostname:      localHostname,
						PeerHostname:       peerHostname,
						LocalOuterIp:       localOuterIp,
						LocalInnerIp:       utils.GetInnerToOuterIp(localOuterIp),
						PeerOuterIp:        peerOuterIp,
						PeerInnerIp:        utils.GetInnerToOuterIp(peerOuterIp),
						PeerPublicKey:      peerPublicKey,
						LocalPrivateKey:    string(localPrivateKey),
						LocalOuterPort:     10000,
						PeerOuterPort:      10000,
						LocalInterfaceName: "wg0",
					})
				} else {
					klog.V(5).Info("Peer node deleted: ", peerHostname)
					err = peerList.Delete(peerHostname)
				}
				if err != nil {
					log.Fatal(err)
				}

				err = utils.UpdateWireguardTunnel(
					*wireguardNamespace,
					"wg0",
					localOuterIp,
					10000,
					utils.GetInnerToOuterIp(localOuterIp),
					string(localPrivateKey),
					peerList)

				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
