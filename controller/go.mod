module github.com/andreaskaris/wireguard-kubernetes/controller

replace github.com/andreaskaris/wireguard-kubernetes/controller => ./

go 1.16

require (
	k8s.io/apimachinery v0.22.3 // indirect
	k8s.io/client-go v0.22.3 // indirect
)
