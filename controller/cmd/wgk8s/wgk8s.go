package main

import (
	"flag"
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/wgk8s"
)

var kubeconfig = flag.String("kubeconfig", "", "Location of kubeconfig file")
var wireguardPrivateKey = flag.String("wg-private-key", "/etc/wireguard/private", "Location of the wireguard private key")
var wireguardPublicKey = flag.String("wg-public-key", "/etc/wireguard/public", "Location of the wireguard public key")
var wireguardNamespace = flag.String("wg-namespace", "wireguard-kubernetes", "Name of the wireguard-kubernetes namespace")
var wireguardInterface = flag.String("wg-interface", "wg0", "Name of the interface inside the wireguard-kubernetes namespace")
var hostname = flag.String("hostname", func() string { s, _ := os.Hostname(); return s }(), "Hostname of this system")
var internalRoutingCidr = flag.String("internal-routing-cidr", "100.64.0.0/16", "Internal routing network used for the wireguard tunnels")

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

	// run this
	wgk8s.Run(clientset,
		*hostname,
		*internalRoutingCidr,
		*wireguardPrivateKey,
		*wireguardPublicKey,
		*wireguardNamespace,
		*wireguardInterface,
	)
}
