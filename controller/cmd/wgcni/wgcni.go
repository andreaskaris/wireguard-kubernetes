package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"

		 "github.com/containernetworking/cni/pkg/types"
	 current "github.com/containernetworking/cni/pkg/types/100"
	 "github.com/containernetworking/cni/pkg/version"

	"github.com/andreaskaris/wireguard-kubernetes/controller/wireguard"
)

var sockaddr = flag.String("sockaddr", "/var/run/wgk8s/wgk8s.sock", "Location wgk8s socket (for CNI communication)")

func main() {
	flag.Parse()

	skel.PluginMain(cmdAdd, cmdDel, version.All)
}

func cmdAdd(args *skel.CmdArgs) error {
	// determine spec version to use
	var netConf struct {
		types.NetConf
		// other plugin-specific configuration goes here
	}
	err := json.Unmarshal(args.StdinData, &netConf)
	cniVersion := netConf.CNIVersion

	// plugin does its work...
	//   set up interfaces
	//   assign addresses, etc
	client, err := rpc.Dial("unix", *sockaddr)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	peerList, err := GetPeers(client)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(peerList)
	// end of plugin work

	// construct the result
	result := &current.Result{
		Interfaces: []*current.Interface{ ... },
		IPs: []*current.IPs{ ... },
		...
	}
	
	// print result to stdout, in the format defined by the requested cniVersion
	return types.PrintResult(result, cniVersion)
}


func SetPeerKeys(client *rpc.Client) error {
	// Synchronous call
	args := wireguard.SetKeysRpcArgs{
		HostIP:       net.ParseIP("172.18.0.4"),
		EndpointPort: 10000,
		PrivateKey:   "privKey",
		PublicKey:    "pubKey",
	}
	var reply int
	err := client.Call("PeerList.SetKeysRpc", args, &reply)
	if err != nil {
		return fmt.Errorf("SetKeysRpc error:", err)
	}
	return nil
}

func GetPeers(client *rpc.Client) (*wireguard.PeerList, error) {
	// Synchronous call
	var peerList wireguard.PeerList
	err := client.Call("PeerList.GetPeersRpc", &struct{}{}, &peerList)
	if err != nil {
		return nil, fmt.Errorf("GetPeersRpc error:", err)
	}
	return &peerList, nil
}
