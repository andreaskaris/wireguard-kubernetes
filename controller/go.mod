module github.com/andreaskaris/wireguard-kubernetes/controller

replace github.com/andreaskaris/wireguard-kubernetes/controller => ./

go 1.16

require (
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.0.1
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0 // indirect
	github.com/go-openapi/validate v0.19.5 // indirect
	github.com/gophercloud/gophercloud v0.1.0 // indirect
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.22.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b // indirect
	sigs.k8s.io/controller-runtime v0.10.3 // indirect
	sigs.k8s.io/structured-merge-diff/v3 v3.0.0 // indirect
)
