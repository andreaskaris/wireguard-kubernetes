package main

import (
	"context"
	"flag"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig = flag.String("kubeconfig", "", "Location of kubeconfig file")

func main() {
	flag.Parse()

	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)
	pods, _ := clientset.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
	fmt.Printf("There are %d nodes in the cluster\n", len(nodes.Items))

	podsWatcher, _ := clientset.CoreV1().Pods("").Watch(context.TODO(), v1.ListOptions{})
	nodesWatcher, _ := clientset.CoreV1().Nodes().Watch(context.TODO(), v1.ListOptions{})

	for {
		select {
		case event := <-podsWatcher.ResultChan():
			fmt.Println("Pod event: ", event)
		case event := <-nodesWatcher.ResultChan():
			fmt.Println("Node event: ", event)
		}
	}
}
