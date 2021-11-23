package wireguard

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
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
		out, err := utils.RunCommandWithOutput(cmd, "EnsureWireguardKeys")
		if err != nil {
			return err
		}
		err = os.WriteFile(wireguardPrivateKey, out, 0660)
		if err != nil {
			return fmt.Errorf("Error in EnsureWireguardKeys: %v", err)
		}
	}
	if !utils.IsFile(wireguardPublicKey) {
		cmd := "cat " + wireguardPrivateKey + " | " + "wg pubkey"
		out, err := utils.RunCommandWithOutput(cmd, "EnsureWireguardKeys")
		if err != nil {
			return err
		}
		err = os.WriteFile(wireguardPublicKey, out, 0660)
		if err != nil {
			return fmt.Errorf("Error in EnsureWireguardKeys: %v", err)
		}
	}

	return nil
}

// EnsureBridge creates the bridge which joins the pod veth endpoints to the overlay.
// If the bridge exists already, it does nothing.
func EnsureBridge(wireguardNamespace, bridgeName, bridgeIp, bridgeIpNetmask string) error {
	cmd := "ip netns exec " + wireguardNamespace + " ip link ls type bridge"
	out, err := utils.RunCommandWithOutput(cmd, "EnsureBridge")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if fields[1] == bridgeName+":" {
			return nil
		}
	}

	cmds := []string{
		"ip netns exec " + wireguardNamespace + " ip link add  " + bridgeName + " type bridge",
		"ip netns exec " + wireguardNamespace + " ip address add dev " + bridgeName + " " + bridgeIp + "/" + bridgeIpNetmask,
		"ip netns exec " + wireguardNamespace + " ip link set dev " + bridgeName + " up",
	}
	for _, cmd := range cmds {
		err := utils.RunCommand(cmd, "EnsureBridge")
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureNamespace creates a namespace with a given name only if the namespace does not exist yet.
// Otherwise, it does nothing.
func EnsureNamespace(wireguardNamespace string) error {
	cmd := "ip netns"
	out, err := utils.RunCommandWithOutput(cmd, "EnsureNamespace")
	if err != nil {
		return err
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
	err = utils.RunCommand(cmd, "EnsureNamespace")
	if err != nil {
		return err
	}

	cmd = "ip netns exec " + wireguardNamespace + " ip link set dev lo up"
	err = utils.RunCommand(cmd, "EnsureNamespace")
	if err != nil {
		return err
	}

	return nil
}

// DeleteNamespace deletes a namespace with a given name if the namespace exists.
// Otherwise, it does nothing.
func DeleteNamespace(wireguardNamespace string) error {
	cmd := "ip netns"
	out, err := utils.RunCommandWithOutput(cmd, "DeleteNamespace")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if fields[0] == wireguardNamespace {
			cmd = "ip netns del " + wireguardNamespace
			err = utils.RunCommand(cmd, "DeleteNamespace")
			if err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func AddPublicKeyLabel(c kubernetes.Interface, hostName, pubKey string) error {
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

	err = setWireguardTunnelPeerRoutes(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	err = pruneWireguardTunnelPeers(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	err = pruneWireguardTunnelPeerRoutes(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	return nil
}

func setWireguardTunnelPeers(wireguardNamespace string, wireguardInterface string, pl *PeerList) error {
	var err error
	for _, p := range *pl {
		cmd := "ip netns exec " + wireguardNamespace + " wg set " + wireguardInterface + " peer " + p.PeerPublicKey + " allowed-ips " + p.PeerInnerIp.String() + "," + p.PeerPodSubnet + " endpoint " + p.PeerOuterIp.String() + ":" + strconv.Itoa(p.PeerOuterPort)
		err = utils.RunCommand(cmd, "setWireguardTunnelPeers")
		if err != nil {
			klog.V(1).Info(err)
		}
	}
	return nil
}

func setWireguardTunnelPeerRoutes(wireguardNamespace string, wireguardInterface string, pl *PeerList) error {
	var err error
	for _, p := range *pl {
		cmd := "ip netns exec " + wireguardNamespace + " ip route add " + p.PeerPodSubnet + " via " + p.PeerInnerIp.String() + " dev " + wireguardInterface
		err = utils.RunCommand(cmd, "setWireguardTunnelPeers")
		if err != nil {
			klog.V(1).Info(err)
		}
	}
	return nil
}

func pruneWireguardTunnelPeers(wireguardNamespace, wireguardInterface string, pl *PeerList) error {
	var err error
	var configuredPeerPublicKeys []string

	cmd := "ip netns exec " + wireguardNamespace + " wg show " + wireguardInterface + " | awk '/^peer/ {print $2}'"
	out, err := utils.RunCommandWithOutput(cmd, "pruneWireguardTunnelPeers")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		configuredPeerPublicKeys = append(configuredPeerPublicKeys, s.Text())
	}
	for _, configuredPeerPublicKey := range configuredPeerPublicKeys {
		found := false
		for _, p := range *pl {
			if p.PeerPublicKey == configuredPeerPublicKey {
				found = true
				break
			}
		}
		if !found {
			cmd := "ip netns exec " + wireguardNamespace + " wg set " + wireguardInterface + " peer " + configuredPeerPublicKey + " remove"
			err := utils.RunCommand(cmd, "pruneWireguardTunnelPeer")
			if err != nil {
				klog.V(1).Info("Could not prune peer ", configuredPeerPublicKey, ": ", err)
				return err
			}
		}
	}
	return nil
}

func pruneWireguardTunnelPeerRoutes(wireguardNamespace, wireguardInterface string, pl *PeerList) error {
	var err error
	var currentRoutes []string

	cmd := "ip netns exec " + wireguardNamespace + " ip route ls dev " + wireguardInterface + " | grep -v 'proto kernel'"

	out, err := utils.RunCommandWithOutput(cmd, "pruneWireguardTunnelPeerRoutes")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		currentRoutes = append(currentRoutes, s.Text())
	}

	for _, currentRoute := range currentRoutes {
		found := false
		currentSubnet := strings.Fields(currentRoute)[0]
		for _, p := range *pl {
			if p.PeerPodSubnet == currentSubnet {
				found = true
				break
			}
		}
		if !found {
			cmd := "ip netns exec " + wireguardNamespace + " ip route delete " + currentRoute
			err := utils.RunCommand(cmd, "pruneWireguardTunnelPeerRoutes")
			if err != nil {
				klog.V(1).Info("Could not prune route ", currentRoute, ": ", err)
				return err
			}
		}
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
		err := utils.RunCommand(cmd, "createWireguardTunnel")
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteWireguardTunnel(wireguardNamespace string, interfaceName string) error {
	cmd := "ip netns exec " + wireguardNamespace + " ip link del " + interfaceName
	err := utils.RunCommand(cmd, "deleteWireguardTunnel")
	if err != nil {
		return err
	}
	return nil
}

func isWireguardTunnel(wireguardNamespace, wireguardInterface string) (bool, error) {
	cmd := "ip netns exec " + wireguardNamespace + " ip -o a | awk '$2 ~ /^" + wireguardInterface + "$/ {print $2}'"
	out, err := utils.RunCommandWithOutput(cmd, "isWireguardTunnel")
	if err != nil {
		return false, err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		text := s.Text()
		return text == wireguardInterface, nil
	}
	return false, nil
}
