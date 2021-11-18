package wireguard

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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
)

func EnsureWireguardKeys(wireguardPrivateKey, wireguardPublicKey string) error {
	if !utils.IsDir(path.Dir(wireguardPrivateKey)) {
		return fmt.Errorf("Directory for private key %s does not exist", wireguardPrivateKey)

	}
	if !utils.IsDir(path.Dir(wireguardPublicKey)) {
		return fmt.Errorf("Directory for private key %s does not exist", wireguardPrivateKey)

	}
	if !utils.IsFile(wireguardPrivateKey) {
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
	if !utils.IsFile(wireguardPublicKey) {
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

func InitWireguardTunnel(wireguardNamespace string, wireguardInterface string, localOuterIp net.IP, localOuterPort int, localInnerIp net.IP, localPrivateKey string) error {
	tunnelExists, err := isWireguardTunnel(wireguardNamespace, wireguardInterface)
	if err != nil {
		return err
	}

	if tunnelExists {
		err := deleteWireguardTunnel(wireguardNamespace, wireguardInterface)
		if err != nil {
			klog.V(5).Info("Could not delete tunnel endpoint: ", err)
		}
	}
	// add new tunnels, for each peer
	err = createWireguardTunnel(
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
	return nil
}

func UpdateWireguardTunnelPeers(wireguardNamespace string, wireguardInterface string, pl *PeerList) error {
	klog.V(5).Info("Updating wireguard tunnels with peer list: ", *pl)
	err := setWireguardTunnelPeers(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	err = pruneWireguardTunnelPeers(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	return nil
}

func setWireguardTunnelPeers(wireguardNamespace string, wireguardInterface string, pl *PeerList) error {
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

func pruneWireguardTunnelPeers(wireguardNamespace, wireguardInterface string, pl *PeerList) error {
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

func createWireguardTunnel(wireguardNamespace string, wireguardInterface string, localOuterIp net.IP, localOuterPort int, localInnerIp net.IP, localPrivateKey string) error {
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

func deleteWireguardTunnel(wireguardNamespace string, interfaceName string) error {
	cmd := "ip netns exec " + wireguardNamespace + " ip link del " + interfaceName
	klog.V(5).Info("Running command: ", cmd)
	err := exec.Command("bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("Error in DeleteWireguardTunnel: %v", err)
	}
	return nil
}

func isWireguardTunnel(wireguardNamespace, wireguardInterface string) (bool, error) {
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
