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
	"regexp"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
)

// EnsureWireguardKeys creates a private key and public key for wireguard, if these keys do not yet exist.
// If the directory (path.Dir) for the key(s) does not exist, this function will throw an error.
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
		"ip netns exec " + wireguardNamespace + " ip link add " + bridgeName + " type bridge",
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
func EnsureNamespace(wireguardNamespace, nodeDefaultInterface string) error {
	err := createNamespace(wireguardNamespace)
	if err != nil {
		return err
	}

	err = connectNamespace(
		wireguardNamespace,
		"to-wg-ns",
		"to-default-ns",
		"169.254.0.1",
		"169.254.0.2",
		"30",
		"eth0",
	)
	if err != nil {
		return err
	}

	return nil
}

// EnsureNamespace creates a namespace with a given name only if the namespace does not exist yet.
// Otherwise, it does nothing.
func connectNamespace(wireguardNamespace, toWireguardNsInterface, toDefaultNsInterface, toWireguardNsInterfaceIp, toDefaultNsInterfaceIp, privateLinkNetmask, nodeDefaultInterface string) error {
	cmd := "ip link ls"
	out, err := utils.RunCommandWithOutput(cmd, "connectNamespace")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		// to-wg-ns@if6:
		if matched, err := regexp.MatchString(`^`+toWireguardNsInterface+`@.*`, fields[1]); err != nil {
			return err
		} else if matched {
			return nil
		}
	}

	cmds := []string{
		"ip link add name " + toWireguardNsInterface + " type veth peer name " + toDefaultNsInterface,
		"ip link set dev " + toDefaultNsInterface + " netns " + wireguardNamespace + "",
		"ip address add dev " + toWireguardNsInterface + " " + toWireguardNsInterfaceIp + "/" + privateLinkNetmask,
		"ip link set dev " + toWireguardNsInterface + " up",
		"ip netns exec " + wireguardNamespace + " ip address add dev " + toDefaultNsInterface + " " + toDefaultNsInterfaceIp + "/" + privateLinkNetmask,
		"ip netns exec " + wireguardNamespace + " ip link set dev " + toDefaultNsInterface + " up",
		"ip netns exec " + wireguardNamespace + " ip route add default via " + toWireguardNsInterfaceIp + " dev " + toDefaultNsInterface + "",
		"ip netns exec " + wireguardNamespace + " iptables -t nat -I POSTROUTING -o " + toDefaultNsInterface + " -j MASQUERADE",
		"ip netns exec " + wireguardNamespace + " iptables -t nat -I POSTROUTING --src " + toWireguardNsInterfaceIp + " -j MASQUERADE",
		"iptables -t nat -I POSTROUTING -o " + nodeDefaultInterface + " --src " + toDefaultNsInterfaceIp + " -j MASQUERADE",
	}
	for _, cmd := range cmds {
		err = utils.RunCommand(cmd, "connectNamespace")
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureNamespace creates a namespace with a given name only if the namespace does not exist yet.
// Otherwise, it does nothing.
func createNamespace(wireguardNamespace string) error {
	cmd := "ip netns"
	out, err := utils.RunCommandWithOutput(cmd, "createNamespace")
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
	err = utils.RunCommand(cmd, "createNamespace")
	if err != nil {
		return err
	}

	cmd = "ip netns exec " + wireguardNamespace + " ip link set dev lo up"
	err = utils.RunCommand(cmd, "createNamespace")
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

// GetNodeTunnelInnerIp returns the IP address that's stored in annotation `wireguard.kubernetes.io/tunnel-ip`.
/*func GetNodeTunnelInnerIp(clientset kubernetes.Interface, hostname string) (net.IP, error) {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), hostname, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	nodeAnnotations := node.GetAnnotations()
	tunnelIp, ok := nodeAnnotations["wireguard.kubernetes.io/tunnel-ip"]
	if ok {
		return net.ParseIP(tunnelIp), nil
	}
	return nil, fmt.Errorf("Could not find annotation '%s' for node %s", "wireguard.kubernetes.io/tunnel-ip", hostname)
}*/

// NodeTunnelInnerIp will either return this node's IP tunnel IP address from the node annotation. Or,
// in absence of such an annotation, it will create a new IP address inside the internalRoutingCidr, it will then
// create the annotation and return the IP
/*func NodeTunnelInnerIp(clientset kubernetes.Interface, localHostname, internalRoutingCidr string) (net.IP, error) {
	tunnelIp, err := GetNodeTunnelInnerIp(clientset, localHostname)

	if err == nil {
		return tunnelIp, nil
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	usedTunnelIps := map[string]bool{}
	for _, node := range nodes.Items {
		nodeAnnotations := node.GetAnnotations()
		tunnelIp, ok := nodeAnnotations["wireguard.kubernetes.io/tunnel-ip"]
		if ok {
			usedTunnelIps[tunnelIp] = true
		}
	}

	// naive retry .. todo
	for i := 0; i < 20; i++ {
		randomIp, err := utils.RandomIpInSubnet(internalRoutingCidr)
		if err != nil {
			return nil, err
		}
		if _, ok := usedTunnelIps[randomIp.String()]; !ok {
			err := PatchNodeAnnotation(clientset, localHostname, "wireguard.kubernetes.io/tunnel-ip", randomIp.String())
			if err != nil {
				return nil, err
			}
			return randomIp, nil
		}
	}

	return nil, fmt.Errorf("Could not find an internal tunnel IP after 20 retries")
}*/

// PatchNodeAnnotation allows to set an annotation on a given node.
func PatchNodeAnnotation(c kubernetes.Interface, hostName, label, value string) error {
	patch := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{{
		Op:    "replace",
		Path:  "/metadata/annotations/" + strings.Replace(label, "/", "~1", -1),
		Value: value,
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

// AddPublicKeyLabel is a wrapper around PatchNodeAnnotation. It adds the public key as an annotation to a host.
func AddPublicKeyLabel(c kubernetes.Interface, hostName, pubKey string) error {
	pubKey = strings.TrimSuffix(pubKey, "\n")
	return PatchNodeAnnotation(c, hostName, "wireguard.kubernetes.io/publickey", pubKey)
}

// InitWireguardTunnel creates a new wireguard tunnel. It first deletes the existing tunnel, then it creates a new tunnel.
// Todo: tunnel deletion is overly aggressive and whenever this process restarts, pod traffic would be interrupted.
func InitWireguardTunnel(wireguardNamespace string, wireguardInterface string, localOuterPort int, localInnerIp net.IP, localPrivateKey string) error {
	tunnelExists, err := isWireguardTunnel(wireguardNamespace, wireguardInterface)
	if err != nil {
		return err
	}

	if tunnelExists {
		err := deleteWireguardTunnel(wireguardNamespace, wireguardInterface)
		if err != nil {
			return fmt.Errorf("Could not delete tunnel endpoint: %s", err)
		}
	}
	// add new tunnels, for each peer
	err = createWireguardTunnel(
		wireguardNamespace,
		wireguardInterface,
		localOuterPort,
		localInnerIp,
		localPrivateKey,
	)
	if err != nil {
		return err
	}
	return nil
}

// UpdateWireguardTunnelPeers applied the contents of pl *PeerList to the wireguard tunnel. Dead routes and peers will be pruned.
func UpdateWireguardTunnelPeers(wireguardNamespace string, wireguardInterface string, pl *PeerList, localPodCidr string) error {
	klog.V(5).Info("Updating wireguard tunnels with peer list: ", *pl)
	err := setWireguardTunnelPeers(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	err = setWireguardTunnelPeerRoutes(wireguardNamespace, wireguardInterface, pl)
	if err != nil {
		return err
	}

	err = setWireguardNamespaceRoutes("to-wg-ns", "169.254.0.2", pl, localPodCidr)
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

	err = pruneWireguardNamespaceRoutes("to-wg-ns", "169.254.0.2", pl, localPodCidr)
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
		err = utils.RunCommand(cmd, "setWireguardTunnelPeerRoutes")
		if err != nil {
			klog.V(1).Info(err)
		}
	}
	return nil
}

func setWireguardNamespaceRoutes(toWireguardNsInterface, toWireguardNsInterfaceIp string, pl *PeerList, localPodCidr string) error {
	var err error
	ips := []string{
		localPodCidr,
	}
	for _, p := range *pl {
		ips = append(ips, p.PeerPodSubnet)
	}
	for _, ip := range ips {
		cmd := "ip route add " + ip + " via " + toWireguardNsInterfaceIp + " dev " + toWireguardNsInterface + ""
		err = utils.RunCommand(cmd, "setWireguardNamespaceRoutes")
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

	cmd := "ip netns exec " + wireguardNamespace + " ip route ls dev " + wireguardInterface

	out, err := utils.RunCommandWithOutput(cmd, "pruneWireguardTunnelPeerRoutes")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		if matched, _ := regexp.Match(".*proto kernel.*", []byte(s.Text())); matched {
			continue
		}
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

func pruneWireguardNamespaceRoutes(toWireguardInterface, toWireguardInterfaceIp string, pl *PeerList, localPodCidr string) error {
	var err error
	var currentRoutes []string
	ips := []string{
		localPodCidr,
	}
	for _, p := range *pl {
		ips = append(ips, p.PeerPodSubnet)
	}

	cmd := "ip route ls dev " + toWireguardInterface

	out, err := utils.RunCommandWithOutput(cmd, "pruneWireguardNamespaceRoutes")
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		if matched, _ := regexp.Match(".*proto kernel.*", []byte(s.Text())); matched {
			continue
		}
		currentRoutes = append(currentRoutes, s.Text())
	}

	for _, currentRoute := range currentRoutes {
		found := false
		currentSubnet := strings.Fields(currentRoute)[0]
		for _, ip := range ips {
			if ip == currentSubnet {
				found = true
				break
			}
		}
		if !found {
			cmd := "ip route delete " + currentRoute
			err := utils.RunCommand(cmd, "pruneWireguardNamespaceRoutes")
			if err != nil {
				klog.V(1).Info("Could not prune route ", currentRoute, ": ", err)
				return err
			}
		}
	}

	return nil
}

func createWireguardTunnel(wireguardNamespace string, wireguardInterface string, localOuterPort int, localInnerIp net.IP, localPrivateKey string) error {
	var cmds []string = []string{
		"ip link add " + wireguardInterface + " type wireguard",
		"wg set " + wireguardInterface + " private-key " + localPrivateKey + " listen-port " + strconv.Itoa(localOuterPort),
		"ip link set dev " + wireguardInterface + " netns " + wireguardNamespace,

		"ip netns exec " + wireguardNamespace + " ip link set dev " + wireguardInterface + " up",
		"ip netns exec " + wireguardNamespace + " ip address add dev " + wireguardInterface + " " + localInnerIp.String() + "/16",
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
	cmd := "ip netns exec " + wireguardNamespace + " ip -o a"

	out, err := utils.RunCommandWithOutput(cmd, "isWireguardTunnel")
	if err != nil {
		return false, err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if fields[1] == wireguardInterface {
			return true, nil
		}
	}
	return false, nil
}
