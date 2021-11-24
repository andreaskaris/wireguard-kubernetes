package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// RunCommand runs a command in the local shell. Returns error on failure.
var RunCommand = func(cmd string, methodName string) error {
	klog.V(5).Info("Running command: ", cmd)
	err := exec.Command("bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in %s: %v (%s)", methodName, err, cmd)
	}
	return nil
}

// RunCommandWithOutput runs a command in the local shell. Returns the stdout or error on failure.
var RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
	klog.V(5).Info("Running command: ", cmd)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return []byte{}, fmt.Errorf("Error in %s: %v (%s) (%s)", methodName, err, cmd, out)
	}
	return out, nil
}

// IsDir returns true if parameter dirname contains the path to a directory.
func IsDir(dirname string) bool {
	fi, err := os.Stat(dirname)
	if err == nil && fi.IsDir() {
		return true
	}
	return false
}

// IsFile returns true if parameter dirname contains the path to a file.
func IsFile(filename string) bool {
	fi, err := os.Stat(filename)
	if err == nil && !fi.IsDir() {
		return true
	}
	return false
}

// GetNodeMachineNetworkIp returns the node's physical IP address (identified as kubernetes type NodeInternalIP).
func GetNodeMachineNetworkIp(node *corev1.Node) (net.IP, error) {
	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeInternalIP {
			return net.ParseIP(a.Address), nil
		}
	}
	return nil, fmt.Errorf("Could not determine machine network IP for node %v", *node)
}

// GetInnterToOuterIp returns the tunnel inner IP address for this node, based on its outer (machine network) IP address.
// TODO: overly simplistic, only works with a /16 machine network at the moment and only IPv4
// This also won't work with remote nodes with overlapping IP addresses
func GetInnerToOuterIp(outerIp net.IP, internalRoutingNet net.IPNet) net.IP {
	return net.IPv4(
		internalRoutingNet.IP[len(internalRoutingNet.IP)-4],
		internalRoutingNet.IP[len(internalRoutingNet.IP)-3],
		outerIp[len(outerIp)-2],
		outerIp[len(outerIp)-1],
	)
}

/*
// todo - real random Ip generator
func RandomIpInSubnet(cidr string) (net.IP, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	ip4 := ip.To4()

	return net.IPv4(
		ip4[0],
		ip4[1],
		ip4[2],
		byte(rand.Intn(255)),
	), nil
}*/

// GetPodCidr returns the IPv4 and/or IPv6 Cidr of a given node.
func GetPodCidr(node *corev1.Node) (map[string]string, error) {
	ips := map[string]string{
		"ipv4": "",
		"ipv6": "",
	}
	if len(node.Spec.PodCIDRs) > 0 {
		for _, cidr := range node.Spec.PodCIDRs {
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, err
			}
			if ip.To4() != nil {
				ips["ipv4"] = cidr
			} else {
				ips["ipv6"] = cidr
			}
		}
	} else {
		ip, _, err := net.ParseCIDR(node.Spec.PodCIDR)
		if err != nil {
			return nil, err
		}
		if ip.To4() != nil {
			ips["ipv4"] = node.Spec.PodCIDR
		} else {
			ips["ipv6"] = node.Spec.PodCIDR
		}
	}
	return ips, nil
}

// GetFirstNetworkAddress returns the local network's first address.
func GetFirstNetworkAddress(cidr string) (string, string, error) {
	s := strings.Split(cidr, "/")
	if len(s) != 2 {
		return "", "", fmt.Errorf("Cannot parse %s", cidr)
	}
	mask := s[1]

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}

	nextIp := ipnet.IP
	nextIp[len(nextIp)-1]++

	return nextIp.String(), mask, nil
}

// GeneratePeerInterfaceName returns the name.
func GenerateVethName(containerId string) string {
	return fmt.Sprintf("veth%s", containerId[:11])
}

// GeneratePeerInterfaceName returns the namespace name, given a full path in /var/run/netns.
func GetNamespaceNameFromPath(fqns string) string {
	return strings.TrimPrefix(fqns, "/var/run/netns/")
}

// GeneratePeerInterfaceName returns the namespace name, given a full path in /var/run/netns.
func GetPathFromNamespace(namespace string) string {
	return "/var/run/netns/" + namespace
}

// GetInterfaceMac returns an interface's MAC address.
func GetInterfaceMac(namespace, interfaceName string) (string, error) {
	cmd := "ip netns exec " + namespace + " ip link ls dev " + interfaceName
	out, err := RunCommandWithOutput(cmd, "GetInterfaceMac")
	if err != nil {
		return "", err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if fields[0] == "link/ether" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("Could not find mac address")
}
