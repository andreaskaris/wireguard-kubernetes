#!/bin/bash

echo "Building cni plugins"
git clone https://github.com/containernetworking/plugins /tmp/plugins
pushd /tmp/plugins
./build_linux.sh
popd

go build -o /tmp/wgcni wgcni.go
if [ "$?" != "0" ]; then
	exit 1
fi

ip netns add fake-pod 2>/dev/null
ip netns add wireguard-kubernetes 2>/dev/null

if [ "$1" == "ADD" ] ; then
export CNI_PATH=/tmp/plugins/bin:/opt/cni/bin:/usr/libexec/cni
export CNI_ARGS="K8S_POD_NAMESPACE=default;K8S_POD_NAME=fedora-deployment-959b9d459-jsssl;K8S_POD_INFRA_CONTAINER_ID=3e8c403e055f61e58afe2f49b4c3d9fa2225763e997b889ba8502ecdcefca1b0;IgnoreUnknown=1"
export CNI_CONTAINERID=3e8c403e055f61e58afe2f49b4c3d9fa2225763e997b889ba8502ecdcefca1b0
export CNI_IFNAME=eth0
export CNI_COMMAND=ADD
export CNI_NETNS=/var/run/netns/fake-pod

elif [ "$1" == "DEL" ] ; then
export CNI_PATH=/tmp/plugins/bin:/opt/cni/bin:/usr/libexec/cni
export CNI_ARGS="IgnoreUnknown=1;K8S_POD_NAMESPACE=default;K8S_POD_NAME=fedora-deployment-959b9d459-jsssl;K8S_POD_INFRA_CONTAINER_ID=3e8c403e055f61e58afe2f49b4c3d9fa2225763e997b889ba8502ecdcefca1b0"
export CNI_CONTAINERID=3e8c403e055f61e58afe2f49b4c3d9fa2225763e997b889ba8502ecdcefca1b0
export CNI_IFNAME=eth0
export CNI_COMMAND=DEL
export CNI_NETNS=/var/run/netns/fake-pod

else
	echo "Must select ADD or DEL"
fi

PLUGIN_CONFIG='{
	"cniVersion": "0.3.1",
	"name": "wgcni",
	"type":"wgcni",
	"ipam": {
		"type": "host-local",
		"dataDir": "/run/cni-ipam-state",
		"routes": [
				{ "dst": "0.0.0.0/0" }
		],
		"ranges": [
			[ { "subnet": "10.244.0.0/24" } ]
		]
	},
	"mtu": 1500
}'

echo "$PLUGIN_CONFIG" | /tmp/wgcni

echo ""
echo ""
echo "=== RESULT ==="
ip netns exec wireguard-kubernetes ip a
ip netns exec fake-pod ip a
grep "" /run/cni-ipam-state/wgcni/*

rm -f /tmp/wgcni
