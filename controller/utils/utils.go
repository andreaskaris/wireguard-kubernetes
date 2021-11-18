package utils

import (
	"fmt"
	"net"
	"os"
	"path"

	corev1 "k8s.io/api/core/v1"
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

func GetInnerToOuterIp(outerIp net.IP) net.IP {
	return net.IPv4(10, 0, 0, outerIp[len(outerIp)-1])
}
