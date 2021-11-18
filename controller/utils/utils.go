package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/andreaskaris/wireguard-kubernetes/controller/wireguard"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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

func EnsureWireguardKeys(wireguardPrivateKey, wireguardPublicKey string) error {
	if !IsDir(path.Dir(wireguardPrivateKey)) {
		return fmt.Errorf("Directory for private key %s does not exist", wireguardPrivateKey)

	}
	if !IsDir(path.Dir(wireguardPublicKey)) {
		return fmt.Errorf("Directory for private key %s does not exist", wireguardPrivateKey)

	}
	if !IsFile(wireguardPrivateKey) {
		cmd := "wg genkey"
		out, err := exec.Command("/bin/bash", "-c", cmd).Output()
		klog.V(5).Info("Running command: ", cmd)
		if err != nil {
			return fmt.Errorf("Error in EnsureWireguardKeys: %v", err)
		}
		err = os.WriteFile(wireguardPrivateKey, out, 0660)
		if err != nil {
			return fmt.Errorf("Error in EnsureWireguardKeys: %v", err)
		}
	}
	if !IsFile(wireguardPublicKey) {
		cmd := "cat " + wireguardPrivateKey + " | " + "wg pubkey"
		klog.V(5).Info("Running command: ", cmd)
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return fmt.Errorf("Error in EnsureWireguardKeys: %v", err)
		}
		err = os.WriteFile(wireguardPublicKey, out, 0660)
		if err != nil {
			return fmt.Errorf("Error in EnsureWireguardKeys: %v", err)
		}
	}

	return nil
}

func EnsureNamespace(wireguardNamespace string) error {
	cmd := "ip netns"
	klog.V(5).Info("Running command: ", cmd)
	out, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("Error in EnsureNamespace: %v", err)
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if fields[0] == wireguardNamespace {
			return nil
		}
	}

	cmd = "ip netns add " + wireguardNamespace
	klog.V(5).Info("Running command: ", cmd)
	err = exec.Command("/bin/bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in EnsureNamespace: %v", err)
	}

	cmd = "ip netns exec " + wireguardNamespace + " ip link set dev lo up"
	klog.V(5).Info("Running command: ", cmd)
	err = exec.Command("/bin/bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in EnsureNamespace: %v", err)
	}

	return nil
}

func AddPublicKeyLabel(c *kubernetes.Clientset, hostName, pubKey string) error {
	pubKey = strings.TrimSuffix(pubKey, "\n")
	patch := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{{
		Op:    "replace",
		Path:  "/metadata/annotations/wireguard.kubernetes.io~1publickey",
		Value: pubKey,
	}}
	patchBytes, _ := json.Marshal(patch)
	_, err := c.CoreV1().Nodes().Patch(
		context.TODO(),
		hostName,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

func UpdateWireguardTunnel(wireguardNamespace string, wireguardInterface string, localOuterIp net.IP, localOuterPort int, localInnerIp net.IP, localPrivateKey string, pl *wireguard.PeerList) error {
	klog.V(5).Info("Updating wireguard tunnels with peer list: ", *pl)
	// delete all tunnels
	//err := DeleteWireguardTunnel(wireguardNamespace, wireguardInterface)
	//if err != nil {
	//	klog.V(5).Info("Could not delete tunnel endpoint: ", err)
	//}

	tunnelExists, err := IsWireguardTunnel(wireguardNamespace, wireguardInterface)
	if err != nil {
		return err
	}
	if !tunnelExists {
		// add new tunnels, for each peer
		err = CreateWireguardTunnel(
			wireguardNamespace,
			wireguardInterface,
			localOuterIp,
			localOuterPort,
			localInnerIp,
			localPrivateKey,
		)
		if err != nil {
			return err
		}
	}

	err = SetWireguardTunnelPeers(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	err = PruneWireguardTunnelPeers(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	return nil
}

func CreateWireguardTunnel(wireguardNamespace string, wireguardInterface string, localOuterIp net.IP, localOuterPort int, localInnerIp net.IP, localPrivateKey string) error {
	var cmds []string = []string{
		"ip link add " + wireguardInterface + " type wireguard",
		"wg set " + wireguardInterface + " private-key " + localPrivateKey + " listen-port " + strconv.Itoa(localOuterPort),
		"ip link set dev " + wireguardInterface + " netns " + wireguardNamespace,

		"ip netns exec " + wireguardNamespace + " ip link set dev " + wireguardInterface + " up",
		"ip netns exec " + wireguardNamespace + " ip address add dev " + wireguardInterface + " " + localInnerIp.String() + "/24",
	}

	for _, cmd := range cmds {
		klog.V(5).Info("Running command: ", cmd)
		out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error in CreateWireguardTunnel: %v (%s) (%s)", err, cmd, out)
		}
	}
	return nil
}

func SetWireguardTunnelPeers(wireguardNamespace string, wireguardInterface string, pl *wireguard.PeerList) error {
	var err error
	for _, p := range *pl {
		cmd := "ip netns exec " + wireguardNamespace + " wg set " + wireguardInterface + " peer " + p.PeerPublicKey + " allowed-ips " + p.PeerInnerIp.String() + " endpoint " + p.PeerOuterIp.String() + ":" + strconv.Itoa(p.PeerOuterPort)
		klog.V(5).Info("Running command: ", cmd)
		err = exec.Command("bash", "-c", cmd).Run()
		if err != nil {
			return fmt.Errorf("Error in CreateWireguardTunnel: %v (%s)", err, cmd)
		}
	}
	return nil
}

func PruneWireguardTunnelPeers(wireguardNamespace, wireguardInterface string, pl *wireguard.PeerList) error {
	var err error
	var configuredPeers []string

	cmd := "ip netns exec " + wireguardNamespace + " wg show " + wireguardInterface + " | awk '/^peer/ {print $2}'"
	klog.V(5).Info("Running command: ", cmd)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("Error in PruneWireguardTunnelPeers: %v (%s)", err, cmd)
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		configuredPeers = append(configuredPeers, s.Text())
	}
	for _, cp := range configuredPeers {
		found := false
		for _, p := range *pl {
			if p.PeerPublicKey == cp {
				found = true
				break
			}
		}
		if !found {
			err := pruneWireguardTunnelPeer(wireguardNamespace, wireguardInterface, cp)
			if err != nil {
				klog.V(1).Info("Could not prune peer ", cp, ": ", err)
			}
		}
	}
	return nil
}

func pruneWireguardTunnelPeer(wireguardNamespace, wireguardInterface, peerPublicKey string) error {
	cmd := "ip netns exec " + wireguardNamespace + " wg set " + wireguardInterface + " peer " + peerPublicKey + " remove"
	klog.V(5).Info("Running command: ", cmd)
	err := exec.Command("bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in pruneWireguardTunnelPeer: %v (%s)", err, cmd)
	}
	return nil
}

func DeleteWireguardTunnel(wireguardNamespace string, interfaceName string) error {
	cmd := "ip netns exec " + wireguardNamespace + " ip link del " + interfaceName
	klog.V(5).Info("Running command: ", cmd)
	err := exec.Command("bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in DeleteWireguardTunnel: %v", err)
	}
	return nil
}

func IsWireguardTunnel(wireguardNamespace, wireguardInterface string) (bool, error) {
	cmd := "ip netns exec " + wireguardNamespace + " ip -o a | awk '$2 ~ /^" + wireguardInterface + "$/ {print $2}'"
	klog.V(5).Info("Running command: ", cmd)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return false, fmt.Errorf("Error in ListWireguardTunnels: %v", err)
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		text := s.Text()
		return text == wireguardInterface, nil
	}
	return false, nil
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
