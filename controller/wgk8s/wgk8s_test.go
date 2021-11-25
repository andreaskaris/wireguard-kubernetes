package wgk8s

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/testdata"
	"github.com/andreaskaris/wireguard-kubernetes/controller/utils"
	"github.com/andreaskaris/wireguard-kubernetes/controller/wireguard"
)

func TestRun(t *testing.T) {
	var err error
	clientset := fake.NewSimpleClientset()

	klog.InitFlags(nil)
	defer klog.Flush()
	flag.Set("v", "10")
	flag.Parse()

	// create the local node first
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNodeLocal, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}

	// Delete the namespace in order to have a clean slate before testing.
	wireguard.DeleteNamespace("wireguard-kubernetes")

	// run the application in a go routine
	go Run(clientset,
		"worker-local",
		"100.64.0.0/16",
		"/etc/wireguard/private",
		"/etc/wireguard/public",
		"wireguard-kubernetes",
		"wg0",
		"wgb0",
	)

	// sleep for 5 seconds (that should be enough to bring up everything)
	time.Sleep(5 * time.Second)

	// now, add 3 worker nodes
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNode0, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}
	time.Sleep(5 * time.Second)
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNode1, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}
	time.Sleep(5 * time.Second)
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNode2, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}
	time.Sleep(5 * time.Second)

	tests := map[string]func(string) error{
		"ip netns exec wireguard-kubernetes ip route ls dev wg0 | grep -v 'proto kernel'": func(out string) error {
			lines := map[string]struct{}{
				"10.245.3.0/24 via 100.64.0.103": struct{}{},
				"10.245.4.0/24 via 100.64.0.104": struct{}{},
				"10.245.5.0/24 via 100.64.0.105": struct{}{},
			}

			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				trimmed := strings.TrimSpace(scanner.Text())
				_, ok := lines[trimmed]
				if !ok {
					return fmt.Errorf("Could not find line '%s' in expected output %v", trimmed, lines)
				}
				delete(lines, trimmed)
			}

			if len(lines) != 0 {
				return fmt.Errorf("Did not find all expected lines in output. Missing lines are %v", lines)
			}
			return nil
		},
		/*
			interface: wg0
			  public key: yTkDw+oNjDy1gju1L37R85RCeFm6+1w5qoLzOzIBzkw=
			  private key: (hidden)
			  listening port: 10000

			peer: qP+1Sstf6Y0MYBeUtJjWthBMfx8uG1hmK4mz9hOQjGI=
			  endpoint: 172.18.0.103:10000
			  allowed ips: 100.64.0.103/32, 10.245.3.0/24

			peer: KmmEwqKHPxZIE2T1dRW51nj4V45W/0eIDibwEinlmQo=
			  endpoint: 172.18.0.104:10000
			  allowed ips: 100.64.0.104/32, 10.245.4.0/24

			peer: dsrxnDAs1KBvvuGuTxi4cr2i/csK+fFCzaq4mX6Mfj0=
			  endpoint: 172.18.0.105:10000
			  allowed ips: 100.64.0.105/32, 10.245.5.0/24
		*/
		// todo improve this verification
		"ip netns exec wireguard-kubernetes wg": func(out string) error {
			lines := map[string]struct{}{
				"endpoint: 172.18.0.103:10000": struct{}{},
				"endpoint: 172.18.0.104:10000": struct{}{},
				"endpoint: 172.18.0.105:10000": struct{}{},
			}

			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				trimmed := strings.TrimSpace(scanner.Text())
				delete(lines, trimmed)
			}

			if len(lines) != 0 {
				return fmt.Errorf("Did not find all expected lines in output. Missing lines are %v", lines)
			}
			return nil
		},
		/*
					1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
			    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
			    inet 127.0.0.1/8 scope host lo
			       valid_lft forever preferred_lft forever
			    inet6 ::1/128 scope host
			       valid_lft forever preferred_lft forever
			2: wgb0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UNKNOWN group default qlen 1000
			    link/ether 26:77:99:59:05:18 brd ff:ff:ff:ff:ff:ff
			    inet 10.245.6.1/24 scope global wgb0
			       valid_lft forever preferred_lft forever
			    inet6 fe80::2477:99ff:fe59:518/64 scope link
			       valid_lft forever preferred_lft forever
			17: wg0: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN group default qlen 1000
			    link/none
			    inet 100.64.0.106/24 scope global wg0
			       valid_lft forever preferred_lft forever
		*/
		// todo: improve this
		"ip netns exec wireguard-kubernetes ip a": func(out string) error {
			lines := map[string]struct{}{
				"wgb0": struct{}{},
				"wg0":  struct{}{},
			}

			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) > 2 {
					delete(lines, strings.TrimRight(fields[1], ":"))
				}
			}

			if len(lines) != 0 {
				return fmt.Errorf("Did not find all expected interfaces in output. Missing interfaces are %v", lines)
			}
			return nil
		},
	}
	for cmd, expected := range tests {
		out, err := utils.RunCommandWithOutput(cmd, "TestRun")
		if err != nil {
			t.Logf("TestRun(): Verification failed for command:\n%s\twith error:\n%s", cmd, err)
			t.Fail()
			continue
		}
		err = expected(string(out))
		if err != nil {
			t.Logf("TestRun(): Verification failed for command:\n'%s'.\nGot:\n'%s'\n\tbut got error:\n%s",
				cmd,
				out,
				err,
			)
			t.Fail()
			continue
		}
	}
}
