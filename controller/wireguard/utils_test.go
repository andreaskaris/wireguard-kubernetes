package wireguard

import (
	"context"
	"fmt"
	"net"
	"path"
	"testing"

	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/andreaskaris/wireguard-kubernetes/controller/testdata"
)

func TestEnsureWireguardKeys(t *testing.T) {
	tempDir := t.TempDir()

	tcs := []struct {
		pubKey      string
		privKey     string
		expectError bool
	}{
		{
			pubKey:      "public",
			privKey:     "private",
			expectError: false,
		}, {
			pubKey:      "subdir/public",
			privKey:     "subdir/private",
			expectError: true,
		},
	}

	for _, tc := range tcs {
		pubKey := path.Join(tempDir, tc.pubKey)
		privKey := path.Join(tempDir, tc.privKey)

		err := EnsureWireguardKeys(pubKey, privKey)
		if !tc.expectError && err != nil {
			t.Fatal(fmt.Sprintf("EnsureWireguardKeys(%s, %s): Got error %s", pubKey, privKey, err))
		}
		if tc.expectError && err == nil {
			t.Fatal(fmt.Sprintf("EnsureWireguardKeys(%s, %s): Should return an error, but got nil", pubKey, privKey))
		}
	}
}

func TestEnsureBridge(t *testing.T) {
	var commandInput map[string]string
	// mock command
	utils.RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}
		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		return []byte(out), nil
	}
	utils.RunCommand = func(cmd string, methodName string) error {
		out, ok := commandInput[cmd]
		if !ok {
			return fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}
		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		return nil
	}

	tcs := []struct {
		commandInput       map[string]string
		wireguardNamespace string
		bridgeName         string
		bridgeIP           string
		bridgeIpNetmask    string
		errorExpected      bool
	}{
		{
			commandInput: map[string]string{
				"ip netns exec wireguard1 ip link ls type bridge": "",
			},
			wireguardNamespace: "wireguard",
			bridgeName:         "wbr0",
			bridgeIP:           "192.168.123.1",
			bridgeIpNetmask:    "24",
			errorExpected:      true,
		},
		{
			commandInput: map[string]string{
				"ip netns exec wireguard ip link ls type bridge": `13: wbr0: <POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> mtu 1360 qdisc fq_codel state UNKNOWN mode DEFAULT group default qlen 500
    link/none 
    `,
			},
			wireguardNamespace: "wireguard",
			bridgeName:         "wbr0",
			bridgeIP:           "192.168.123.1",
			bridgeIpNetmask:    "24",
			errorExpected:      false,
		},
		{
			commandInput: map[string]string{
				"ip netns exec wireguard ip link ls type bridge":       "",
				"ip netns exec wireguard ip link add wbr0 type bridge": "",
			},
			wireguardNamespace: "wireguard",
			bridgeName:         "wbr0",
			bridgeIP:           "192.168.123.1",
			bridgeIpNetmask:    "24",
			errorExpected:      true,
		},
		{
			commandInput: map[string]string{
				"ip netns exec wireguard ip link ls type bridge":                   "",
				"ip netns exec wireguard ip link add wbr0 type bridge":             "",
				"ip netns exec wireguard ip address add dev wbr0 192.168.123.1/24": "",
				"ip netns exec wireguard ip link set dev wbr0 up":                  "",
			},
			wireguardNamespace: "wireguard",
			bridgeName:         "wbr0",
			bridgeIP:           "192.168.123.1",
			bridgeIpNetmask:    "24",
			errorExpected:      false,
		},
	}

	for k, tc := range tcs {
		commandInput = tc.commandInput
		err := EnsureBridge(tc.wireguardNamespace, tc.bridgeName, tc.bridgeIP, tc.bridgeIpNetmask)
		if tc.errorExpected != (err != nil) {
			t.Fatal(
				fmt.Sprintf(
					"EnsureBridge(%s, %s, %s, %s) - Test %d: Expected to see error: %t. Instead, got: %s",
					tc.wireguardNamespace,
					tc.bridgeName,
					tc.bridgeIP,
					tc.bridgeIpNetmask,
					k,
					tc.errorExpected,
					err,
				),
			)
		}
	}
}

func TestEnsureNamespace(t *testing.T) {
	var commandInput map[string]string
	// mock command
	utils.RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return []byte(out), nil
	}
	utils.RunCommand = func(cmd string, methodName string) error {
		out, ok := commandInput[cmd]
		if !ok {
			return fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return nil
	}

	tcs := []struct {
		commandInput       map[string]string
		wireguardNamespace string
		errorExpected      bool
		mustRunAllCommands bool
	}{
		{
			commandInput: map[string]string{
				"none": "",
			},
			wireguardNamespace: "wireguard",
			errorExpected:      true,
		},
		{
			commandInput: map[string]string{
				"ip netns": `test
wireguard
`,
			},
			wireguardNamespace: "wireguard",
			errorExpected:      false,
		},
		{
			commandInput: map[string]string{
				"ip netns": `test
`,
				"ip netns add wireguard":                        "",
				"ip netns exec wireguard ip link set dev lo up": "",
			},
			wireguardNamespace: "wireguard",
			errorExpected:      false,
			mustRunAllCommands: true,
		},
	}

	for k, tc := range tcs {
		commandInput = tc.commandInput
		err := EnsureNamespace(tc.wireguardNamespace)
		if tc.errorExpected != (err != nil) {
			t.Fatal(
				fmt.Sprintf(
					"EnsureNamespace(%s) - Test %d: Expected to see error: %t. Instead, got: %s",
					tc.wireguardNamespace,
					k,
					tc.errorExpected,
					err,
				),
			)
		}
		if tc.mustRunAllCommands && len(commandInput) > 0 {
			t.Fatalf("EnsureNamespace(%s): Did not run all commands, leftover commands are %v",
				tc.wireguardNamespace,
				commandInput,
			)
		}
	}

}

func TestDeleteNamespace(t *testing.T) {
	var commandInput map[string]string
	// mock command
	utils.RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return []byte(out), nil
	}
	utils.RunCommand = func(cmd string, methodName string) error {
		out, ok := commandInput[cmd]
		if !ok {
			return fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return nil
	}

	tcs := []struct {
		commandInput       map[string]string
		wireguardNamespace string
		errorExpected      bool
		mustRunAllCommands bool
	}{
		{
			commandInput: map[string]string{
				"none": "",
			},
			wireguardNamespace: "wireguard",
			errorExpected:      true,
		},
		{
			commandInput: map[string]string{
				"ip netns": `test
`,
			},
			wireguardNamespace: "wireguard",
			errorExpected:      false,
			mustRunAllCommands: true,
		},
		{
			commandInput: map[string]string{
				"ip netns": `test
wireguard
`,
				"ip netns del wireguard": "",
			},
			wireguardNamespace: "wireguard",
			errorExpected:      false,
			mustRunAllCommands: true,
		},
	}

	for k, tc := range tcs {
		commandInput = tc.commandInput
		err := DeleteNamespace(tc.wireguardNamespace)
		if tc.errorExpected != (err != nil) {
			t.Fatal(
				fmt.Sprintf(
					"DeleteNamespace(%s) - Test %d: Expected to see error: %t. Instead, got: %s",
					tc.wireguardNamespace,
					k,
					tc.errorExpected,
					err,
				),
			)
		}
		if tc.mustRunAllCommands && len(commandInput) > 0 {
			t.Fatalf("DeleteNamespace(%s): Did not run all commands, leftover commands are %v",
				tc.wireguardNamespace,
				commandInput,
			)
		}
	}

}

func TestAddPublicKeyLabel(t *testing.T) {
	var err error
	clientset := fake.NewSimpleClientset()

	localHostname := "worker-local"
	testPubKey := "testPubKey"

	// create the local node first
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNodeLocal, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("TestAddPublicKeyLabel(): Error retrieving node: %s", err)
	}

	err = AddPublicKeyLabel(clientset, localHostname, testPubKey)
	if err != nil {
		t.Fatalf("AddPublicKeyLabel(clientset, %s, %s): Got error %s",
			localHostname,
			testPubKey,
			err,
		)
	}

	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), localHostname, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("TestAddPublicKeyLabel: Cannot retrieve information about local node: %s", err)
	}
	nodeAnnotations := node.GetAnnotations()
	publicKey, ok := nodeAnnotations["wireguard.kubernetes.io/publickey"]
	if !ok {
		t.Fatal("TestAddPublicKeyLabel(): Cannot retrieve public key annotation")
	}
	if publicKey != testPubKey {
		t.Fatalf("TestAddPublicKeyLabel(): Desired value '%s' != retrieved value '%s'",
			testPubKey,
			publicKey,
		)
	}
}

func TestInitWireguardTunnel(t *testing.T) {
	var commandInput map[string]string
	// mock command
	utils.RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return []byte(out), nil
	}
	utils.RunCommand = func(cmd string, methodName string) error {
		out, ok := commandInput[cmd]
		if !ok {
			return fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return nil
	}

	tcs := []struct {
		commandInput       map[string]string
		wireguardNamespace string
		wireguardInterface string
		localOuterPort     int
		localInnerIp       net.IP
		localPrivateKey    string
		errorExpected      bool
		mustRunAllCommands bool
	}{
		{
			commandInput: map[string]string{
				"ip netns exec wireguard ip -o a": `1: lo    inet 127.0.0.1/8 scope host lo\       valid_lft forever preferred_lft forever
1: lo    inet6 ::1/128 scope host \       valid_lft forever preferred_lft forever
2: eth0    inet 192.168.122.79/24 brd 192.168.122.255 scope global dynamic noprefixroute eth0\       valid_lft 3122sec preferred_lft 3122sec
2: eth0    inet6 fe80::7853:68f9:60a6:9953/64 scope link noprefixroute \       valid_lft forever preferred_lft forever
3: wg0    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0\       valid_lft forever preferred_lft forever
4: br-cf0d8ccbf6a5    inet 172.18.0.1/16 brd 172.18.255.255 scope global br-cf0d8ccbf6a5\       valid_lft forever preferred_lft forever
4: br-cf0d8ccbf6a5    inet6 fc00:f853:ccd:e793::1/64 scope global tentative \       valid_lft forever preferred_lft forever
4: br-cf0d8ccbf6a5    inet6 fe80::1/64 scope link tentative \       valid_lft forever preferred_lft forever
`,
				"ip netns exec wireguard ip link del wg0":                    "",
				"ip link add wg0 type wireguard":                             "",
				"wg set wg0 private-key privateKey listen-port 10000":        "",
				"ip link set dev wg0 netns wireguard":                        "",
				"ip netns exec wireguard ip link set dev wg0 up":             "",
				"ip netns exec wireguard ip address add dev wg0 10.0.0.1/24": "",
			},
			wireguardNamespace: "wireguard",
			wireguardInterface: "wg0",
			localOuterPort:     10000,
			localInnerIp:       net.ParseIP("10.0.0.1"),
			localPrivateKey:    "privateKey",
			errorExpected:      false,
			mustRunAllCommands: true,
		},
	}

	for k, tc := range tcs {
		commandInput = tc.commandInput
		err := InitWireguardTunnel(
			tc.wireguardNamespace,
			tc.wireguardInterface,
			tc.localOuterPort,
			tc.localInnerIp,
			tc.localPrivateKey,
		)
		if tc.errorExpected != (err != nil) {
			t.Fatal(
				fmt.Sprintf(
					"InitWireguardTunnel(%s, %s, %d, %s, %s) - Test %d: Expected to see error: %t. Instead, got: %s",
					tc.wireguardNamespace,
					tc.wireguardInterface,
					tc.localOuterPort,
					tc.localInnerIp.String(),
					tc.localPrivateKey,
					k,
					tc.errorExpected,
					err,
				),
			)
		}
		if tc.mustRunAllCommands && len(commandInput) > 0 {
			t.Fatalf("InitWireguardTunnel(%s, %s, %d, %s, %s): Did not run all commands, leftover commands are %v",
				tc.wireguardNamespace,
				tc.wireguardInterface,
				tc.localOuterPort,
				tc.localInnerIp.String(),
				tc.localPrivateKey,
				commandInput,
			)
		}
	}
}

func TestUpdateWireguardTunnelPeers(t *testing.T) {
	var commandInput map[string]string
	// mock command
	utils.RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return []byte(out), nil
	}
	utils.RunCommand = func(cmd string, methodName string) error {
		out, ok := commandInput[cmd]
		if !ok {
			return fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}

		t.Logf("\n\tCommand is:\n%s\n\tOutput is:\n%s", cmd, out)
		delete(commandInput, cmd)

		return nil
	}

	tcs := []struct {
		commandInput       map[string]string
		wireguardNamespace string
		wireguardInterface string
		pl                 PeerList
		errorExpected      bool
		mustRunAllCommands bool
	}{
		{
			commandInput: map[string]string{
				"ip netns exec wireguard wg set wg0 peer peerPublicKey allowed-ips 10.0.0.2,10.244.0.0/24 endpoint 192.168.123.2:10000": "",
				"ip netns exec wireguard ip route add 10.244.0.0/24 via 10.0.0.2 dev wg0":                                               "",
				"ip netns exec wireguard wg show wg0 | awk '/^peer/ {print $2}'": `peerPublicKey
toBePrunedKey`,
				"ip netns exec wireguard wg set wg0 peer toBePrunedKey remove": "",
				"ip netns exec wireguard ip route ls dev wg0 | grep -v 'proto kernel'": `10.244.0.0/24 via 10.0.0.2 
10.245.5.0/24 via 100.64.0.105`,
				"ip netns exec wireguard ip route delete 10.245.5.0/24 via 100.64.0.105": "",
			},
			wireguardNamespace: "wireguard",
			wireguardInterface: "wg0",
			pl: PeerList{
				"peerHostname": &Peer{
					PeerHostname:  "peerHostname",
					PeerOuterIp:   net.ParseIP("192.168.123.2"),
					PeerInnerIp:   net.ParseIP("10.0.0.2"),
					PeerPublicKey: "peerPublicKey",
					PeerOuterPort: 10000,
					PeerPodSubnet: "10.244.0.0/24",
				},
			},
			errorExpected:      false,
			mustRunAllCommands: true,
		},
	}

	for k, tc := range tcs {
		commandInput = tc.commandInput
		err := UpdateWireguardTunnelPeers(
			tc.wireguardNamespace,
			tc.wireguardInterface,
			&tc.pl,
		)
		if tc.errorExpected != (err != nil) {
			t.Fatal(
				fmt.Sprintf(
					"UpdateWireguardTunnelPeers(%s, %s, %v) - Test %d: Expected to see error: %t. Instead, got: %s",
					tc.wireguardNamespace,
					tc.wireguardInterface,
					tc.pl,
					k,
					tc.errorExpected,
					err,
				),
			)
		}
		if tc.mustRunAllCommands && len(commandInput) > 0 {
			t.Fatalf("UpdateWireguardTunnelPeers(%s, %s, %v): Did not run all commands, leftover commands are %v",
				tc.wireguardNamespace,
				tc.wireguardInterface,
				tc.pl,
				commandInput,
			)
		}
	}
}
