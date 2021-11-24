// Copyright 2021 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ipam"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

const (
	wireguardNamespace = "wireguard-kubernetes"
	wireguardBridge    = "wgb0"
)

type NetConf struct {
	types.NetConf
}

type EnvArgs struct {
	types.CommonArgs
	K8S_POD_NAMESPACE          types.UnmarshallableString `json:"k8s_pod_namespace,omitempty"`
	K8S_POD_NAME               types.UnmarshallableString `json:"k8s_pod_name,omitempty"`
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString `json:"k8s_pod_infra_container_id,omitempty"`
}

func main() {
	flag.Parse()

	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("none"))
}

// cmdAdd is run when action ADD is provided.
func cmdAdd(args *skel.CmdArgs) error {
	var success bool = false

	// pass configuration into an NetConf object
	netConf, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	// we rely on IPAM
	if netConf.IPAM.Type == "" {
		return fmt.Errorf("An IPAM plugin must be specified")
	}

	// determine the veth name inside the wireguard-kubernetes namespace
	wireguardInterface := utils.GenerateVethName(args.ContainerID)
	// determine the pod's namespace and interface name
	podNamespace := utils.GetNamespaceNameFromPath(args.Netns)
	podInterface := args.IfName

	// create the veth interface that joins the pod's network with bridge wgb0 inside wireguard-kubernetes
	hostInterface, containerInterface, err := createVeth(podNamespace, podInterface, wireguardNamespace, wireguardInterface, wireguardBridge)
	if err != nil {
		return err
	}

	// start building the result object
	result := &current.Result{
		CNIVersion: current.ImplementedSpecVersion,
		Interfaces: []*current.Interface{
			hostInterface,
			containerInterface,
		},
	}

	// run the IPAM plugin and get back the config to apply
	// we currently require that an IPAM plugin be configured
	r, err := ipam.ExecAdd(netConf.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	// release IP in case of failure
	defer func() {
		if !success {
			ipam.ExecDel(netConf.IPAM.Type, args.StdinData)
		}
	}()

	// Convert whatever the IPAM result was into the current Result type
	ipamResult, err := current.NewResultFromResult(r)
	if err != nil {
		return err
	}

	result.IPs = ipamResult.IPs
	result.Routes = ipamResult.Routes

	if len(result.IPs) == 0 {
		return errors.New("IPAM plugin returned missing IP config")
	}

	// now that IPAM returned our IP addresses and routes, apply them
	err = addIpConfiguration(podNamespace, podInterface, result.IPs, result.Routes)
	if err != nil {
		return err
	}

	success = true

	return types.PrintResult(result, netConf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	// todo - replace all of this with https://github.com/vishvananda/netlink
	netConf, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	// we rely on IPAM
	if netConf.IPAM.Type == "" {
		return fmt.Errorf("An IPAM plugin must be specified")
	}

	wireguardInterface := utils.GenerateVethName(args.ContainerID)
	podNamespace := utils.GetNamespaceNameFromPath(args.Netns)
	podInterface := args.IfName

	err = deleteVeth(podNamespace, podInterface, wireguardNamespace, wireguardInterface)
	if err != nil {
		return err
	}

	ipam.ExecDel(netConf.IPAM.Type, args.StdinData)

	return types.PrintResult(&current.Result{}, netConf.CNIVersion)
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func loadNetConf(data []byte) (*NetConf, string, error) {
	conf := &NetConf{}
	if err := json.Unmarshal(data, &conf); err != nil {
		return nil, "", fmt.Errorf("failed to parse")
	}

	return conf, conf.CNIVersion, nil
}

func loadArgs(args *skel.CmdArgs) (*EnvArgs, error) {
	envArgs := EnvArgs{}
	if err := types.LoadArgs(args.Args, &envArgs); err != nil {
		return nil, err
	}
	return &envArgs, nil
}

// createVeth creates the veth pair for this pod, with one end inside the wireguard-kubernetes namespace, and the
// other end inside the pod as eth0.
func createVeth(podNamespace, podInterface, wireguardNamespace, wireguardInterface, wireguardBridge string) (*current.Interface, *current.Interface, error) {
	// todo - replace all of this with https://github.com/vishvananda/netlink
	cmds := []string{
		"ip netns exec " + podNamespace + " ip link add name " + podInterface + " type veth peer name " + wireguardInterface,
		"ip netns exec " + podNamespace + " ip link set dev " + podInterface + " up",
		"ip netns exec " + podNamespace + " ip link set dev " + wireguardInterface + " netns " + wireguardNamespace,
		"ip netns exec " + wireguardNamespace + " ip link set dev " + wireguardInterface + " master " + wireguardBridge,
		"ip netns exec " + wireguardNamespace + " ip link set dev " + wireguardInterface + " up",
	}
	for _, cmd := range cmds {
		err := utils.RunCommand(cmd, "cmdAdd")
		if err != nil {
			return nil, nil, err
		}
	}

	wireguardInterfaceMac, err := utils.GetInterfaceMac(wireguardNamespace, wireguardInterface)
	if err != nil {
		return nil, nil, err
	}
	hostInterface := current.Interface{
		Name:    wireguardInterface,
		Mac:     wireguardInterfaceMac,
		Sandbox: utils.GetPathFromNamespace(wireguardNamespace),
	}
	containerInterfaceMac, err := utils.GetInterfaceMac(podNamespace, podInterface)
	if err != nil {
		return nil, nil, err
	}
	containerInterface := current.Interface{
		Name:    podInterface,
		Mac:     containerInterfaceMac,
		Sandbox: utils.GetPathFromNamespace(podNamespace),
	}
	return &hostInterface, &containerInterface, nil
}

// deleteVeth deletes the given veth pair (only one side must be deleted)
func deleteVeth(podNamespace, podInterface, wireguardNamespace, wireguardInterface string) error {
	// todo - replace all of this with https://github.com/vishvananda/netlink
	cmds := []string{
		"ip netns exec " + podNamespace + " ip link del " + podInterface,
	}
	for _, cmd := range cmds {
		err := utils.RunCommand(cmd, "cmdAdd")
		if err != nil {
			return err
		}
	}
	return nil
}

// addIpConfiguration sets up this interface's IP address and routes for the pod.
func addIpConfiguration(podNamespace, podInterface string, ips []*current.IPConfig, routes []*types.Route) error {
	cmds := []string{}
	for _, ip := range ips {
		cmds = append(
			cmds,
			"ip netns exec "+podNamespace+" ip address add dev "+podInterface+" "+ip.Address.String(),
		)

		for _, route := range routes {
			cmds = append(
				cmds,
				"ip netns exec "+podNamespace+" ip route add "+route.Dst.String()+" via "+ip.Gateway.String()+" dev "+podInterface,
			)
		}
	}
	for _, cmd := range cmds {
		err := utils.RunCommand(cmd, "addIpConfiguration")
		if err != nil {
			return err
		}
	}

	return nil
}
