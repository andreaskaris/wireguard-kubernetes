package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// PrepareSocketPath prepares the directory for the socket by creating the directory
// if it does not exist and by removing any existing socket file.
func PrepareSocketPath(sockaddr string) error {
	// Create directory if needed.
	err := os.MkdirAll(path.Dir(sockaddr), 0775)
	if err != nil {
		return fmt.Errorf("Error in PrepareSocketPath: %v", err)
	}

	// Remove the socket if there is one.
	if err := os.RemoveAll(sockaddr); err != nil {
		return fmt.Errorf("Error in PrepareSocketPath: %v", err)
	}

	return nil
}

func IsDir(dirname string) bool {
	fi, err := os.Stat(dirname)
	if err == nil && fi.IsDir() {
		return true
	}
	return false
}

func IsFile(filename string) bool {
	fi, err := os.Stat(filename)
	if err == nil && !fi.IsDir() {
		return true
	}
	return false
}

func GetNodeInternalIp(node *corev1.Node) (net.IP, error) {
	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeInternalIP {
			return net.ParseIP(a.Address), nil
		}
	}
	return nil, fmt.Errorf("Could not determine node internal IP for node %v", *node)
}

// TODO: overly simplistic, only works with a /16 machine network at the moment and only IPv4
func GetInnerToOuterIp(outerIp net.IP, internalRoutingNet net.IPNet) net.IP {
	return net.IPv4(
		internalRoutingNet.IP[len(internalRoutingNet.IP)-4],
		internalRoutingNet.IP[len(internalRoutingNet.IP)-3],
		outerIp[len(outerIp)-2],
		outerIp[len(outerIp)-1],
	)
}

// GetPodCidr returns the IPv4 and/or IPv6 Cidr of a given node
func GetPodCidr(node *corev1.Node) map[string]string {
	ips := map[string]string{
		"ipv4": "",
		"ipv6": "",
	}
	if len(node.Spec.PodCIDRs) > 0 {
		for _, cidr := range node.Spec.PodCIDRs {
			if net.ParseIP(cidr).To4() == nil {
				ips["ipv4"] = cidr
			} else {
				ips["ipv6"] = cidr
			}
		}
	} else {
		if net.ParseIP(node.Spec.PodCIDR).To4() == nil {
			ips["ipv4"] = node.Spec.PodCIDR
		} else {
			ips["ipv6"] = node.Spec.PodCIDR
		}
	}
	return ips
}

var RunCommand = func(cmd string, methodName string) error {
	klog.V(5).Info("Running command: ", cmd)
	err := exec.Command("bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in %s: %v (%s)", methodName, err, cmd)
	}
	return nil
}

var RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
	klog.V(5).Info("Running command: ", cmd)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return []byte{}, fmt.Errorf("Error in %s: %v (%s) (%s)", methodName, err, cmd, out)
	}
	return out, nil
}
