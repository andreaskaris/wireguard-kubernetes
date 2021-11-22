package wgk8s

import (
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	 "k8s.io/klog"

	"github.com/andreaskaris/wireguard-kubernetes/controller/testdata"
)

func TestRun(t *testing.T) {
	var err error
	clientset := fake.NewSimpleClientset()

	klog.InitFlags(nil)
        defer klog.Flush()
	flag.Set("v", "10")
        flag.Parse()

	// create the local node first
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNode0, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}

	// run the application in a go routine
	go Run(clientset,
                "worker-0",
                "100.64.0.0/16",
                "/etc/wireguard/private",
                "/etc/wireguard/public",
                "wireguard-kubernetes",
                "wg0",
        )

	// sleep for 5 seconds (that should be enough to bring up everything)
	time.Sleep(5 * time.Second)

	// now, add 2 worker nodes
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNode1, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}
	time.Sleep(5 * time.Second)
	_, err = clientset.CoreV1().Nodes().Create(context.TODO(), testdata.WorkerNode2, metav1.CreateOptions{})
	if err != nil {
		fmt.Print(err.Error())
	}

	// and sleep for another minute
	time.Sleep(60 * time.Second)
}
