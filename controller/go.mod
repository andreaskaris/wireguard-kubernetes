module github.com/andreaskaris/wireguard-kubernetes/controller

replace github.com/andreaskaris/wireguard-kubernetes/controller => ./

go 1.16

require (
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.0.1
	github.com/imdario/mergo v0.3.12 // indirect
	golang.org/x/sys v0.0.0-20210817190340-bfb29a6856f2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.22.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b // indirect
)
