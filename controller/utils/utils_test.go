package utils

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/andreaskaris/wireguard-kubernetes/controller/testdata"
)

func TestIsDir(t *testing.T) {
	tempDir := t.TempDir()

	if IsDir(tempDir) == false {
		t.Fatal("IsDir(tempDir): Expected to be true, not false")
	}
}

func TestIsFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := tempDir + "/file"

	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatal(fmt.Sprintf("TestIsFile(): Failed to create file %s", testFile))
	}
	if IsFile(testFile) == false {
		t.Fatal(fmt.Sprintf("IsFile(%v): Expected to be true, not false", testFile))
	}
}

func TestGetNodeMachineNetworkIp(t *testing.T) {
	ip, err := GetNodeMachineNetworkIp(testdata.MasterNode0)
	if err != nil {
		t.Fatal(fmt.Sprintf("GetNodeMachineNetworkIp(testdata.MasterNode0): Expected to return nil error, instead got %s", err))
	}
	if ip.String() != "172.18.0.100" {
		t.Fatal(fmt.Sprintf("GetNodeMachineNetworkIp(testdata.MasterNode0): Expected to get 172.18.0.100, instead got %s", ip.String()))
	}
}

func TestGetInnerToOuterIp(t *testing.T) {
	tcs := []struct {
		outerIp            string
		internalRoutingNet string
		expected           string
	}{
		{
			outerIp:            "172.18.100.115",
			internalRoutingNet: "10.64.0.0/16",
			expected:           "10.64.100.115",
		},
	}
	for k, tc := range tcs {
		ip := net.ParseIP(tc.outerIp)
		_, ipnet, err := net.ParseCIDR(tc.internalRoutingNet)
		if err != nil {
			t.Fatal(fmt.Sprintf("TestGetInnerToOuterIp().Test%d: Could not parse internalRoutingNet %s, got error %s", k, tc.internalRoutingNet, err))
		}
		out := GetInnerToOuterIp(ip, *ipnet)
		if out.String() != tc.expected {
			t.Fatal(fmt.Sprintf("GetInnerToOuterIp(%s, %s): Expected to get %s, instead got %s", tc.outerIp, tc.internalRoutingNet, tc.expected, out.String()))
		}
	}
}

func TestGetPodCidr(t *testing.T) {
	podCidr, err := GetPodCidr(testdata.MasterNode0)
	if err != nil {
		t.Fatal(fmt.Sprintf("GetPodCidr(testdata.MasterNode0): Expected to return nil error, instead got %s", err))
	}
	if podCidr["ipv4"] != "10.245.0.0/24" {
		t.Fatal(fmt.Sprintf("GetPodCidr(testdata.MasterNode0)[ipv4]: Expected to get 10.245.0.0/24, instead got %s", podCidr["ipv4"]))
	}
	if podCidr["ipv6"] != "2000::3/64" {
		t.Fatal(fmt.Sprintf("GetPodCidr(testdata.MasterNode0)[ipv6]: Expected to get 2000::3/64, instead got %s", podCidr["ipv6"]))
	}
}

func TestGetFirstNetworkAddress(t *testing.T) {
	tcs := []struct {
		cidr string
		ip   string
		mask string
	}{
		{
			cidr: "10.1.0.250/8",
			ip:   "10.0.0.1",
			mask: "8",
		},
		{
			cidr: "2000::3/64",
			ip:   "2000::1",
			mask: "64",
		},
	}
	for _, tc := range tcs {
		ip, mask, err := GetFirstNetworkAddress(tc.cidr)
		if err != nil {
			t.Fatal(fmt.Sprintf("GetFirstNetworkAddress(%s): Expected to return nil error, instead got %s", tc.cidr, err))
		}
		if tc.ip != ip || tc.mask != mask {
			t.Fatal(fmt.Sprintf("GetFirstNetworkAddress(%s): Expected to get %s, %s but instead got %s, %s", tc.cidr, tc.ip, tc.mask, ip, mask))

		}

	}
}

func TestGenerateVethName(t *testing.T) {
	tcs := []struct {
		in  string
		out string
	}{
		{
			in:  "57d2933c-4848-4d13-9656-dd061b6320bf",
			out: "veth57d2933c-48",
		},
	}
	for _, tc := range tcs {
		vethName := GenerateVethName(tc.in)
		if vethName != tc.out {
			t.Fatal(fmt.Sprintf("GenerateVethName(%s): Expected %s, got %s", tc.in, tc.out, vethName))
		}
	}
}

func TestGetNamespaceNameFromPath(t *testing.T) {
	tcs := []struct {
		in  string
		out string
	}{
		{
			in:  "/var/run/netns/testns",
			out: "testns",
		},
	}
	for _, tc := range tcs {
		vethName := GetNamespaceNameFromPath(tc.in)
		if vethName != tc.out {
			t.Fatal(fmt.Sprintf("GetNamespaceNameFromPath(%s): Expected %s, got %s", tc.in, tc.out, vethName))
		}
	}
}

func TestGetPathFromNamespace(t *testing.T) {
	tcs := []struct {
		in  string
		out string
	}{
		{
			in:  "testns",
			out: "/var/run/netns/testns",
		},
	}
	for _, tc := range tcs {
		vethName := GetPathFromNamespace(tc.in)
		if vethName != tc.out {
			t.Fatal(fmt.Sprintf("GetPathFromNamespace(%s): Expected %s, got %s", tc.in, tc.out, vethName))
		}
	}
}

func TestGetInterfaceMac(t *testing.T) {
	// set up mock data
	commandInput := map[string]string{
		"ip netns exec TestGetInterfaceMac ip link ls dev dummy0": `5: dummy0: <BROADCAST,NOARP> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
			    link/ether 00:ab:ab:ab:ab:ab brd ff:ff:ff:ff:ff:ff
			    `,
	}

	// mock command
	RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}
		return []byte(out), nil
	}

	// run the test now
	expectedMac := "00:ab:ab:ab:ab:ab"

	mac, err := GetInterfaceMac("TestGetInterfaceMac", "dummy0")
	if err != nil {
		t.Fatal(fmt.Sprintf("GetInterfaceMac(%s, %s): Expected to return nil error, instead got %s", "TestGetInterfaceMac", "dummy0", err))
	}
	if mac != expectedMac {
		t.Fatal(fmt.Sprintf("GetInterfaceMac(%s, %s): Expected to get mac address %s, instead got %s", "TestGetInterfaceMac", "dummy0", expectedMac, mac))
	}
}

func TestGetInterfaceToIp(t *testing.T) {
	// set up mock data
	commandInput := map[string]string{
		"ip -o address | grep 192.168.122.79": `2: eth0    inet 192.168.122.79/24 brd 192.168.122.255 scope global dynamic noprefixroute eth0\       valid_lft 2209sec preferred_lft 2209sec
		`,
	}

	// mock command
	RunCommandWithOutput = func(cmd string, methodName string) ([]byte, error) {
		out, ok := commandInput[cmd]
		if !ok {
			return []byte{}, fmt.Errorf("Unknown command '%s' in method '%s'", cmd, methodName)
		}
		return []byte(out), nil
	}

	// run the test now
	ipAddress := "192.168.122.79"
	expectedInterface := "eth0"
	intf, err := GetInterfaceToIp(net.ParseIP(ipAddress))
	if err != nil {
		t.Fatal(fmt.Sprintf("GetInterfaceToIp(%s): Expected to return nil error, instead got %s", ipAddress, err))
	}
	if intf != expectedInterface {
		t.Fatal(fmt.Sprintf("GetInterfaceToIp(%s): Expected to get interface name %s, instead got %s", ipAddress, expectedInterface, intf))
	}
}
