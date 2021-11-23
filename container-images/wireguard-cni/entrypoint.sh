#!/bin/bash

PodCIDR=$(kubectl get nodes $(cat /etc/hostname) -o json | jq '.spec.podCIDR' | sed 's/"//g')

cat <<EOF > /etc/cni/net.d/05-wireguard-cni.conf
{
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
			[ { "subnet": "${PodCIDR}" } ]
		]
	},
	"mtu": 1500
}
EOF

for f in /cni-bin/*; do
	cp -n $f /opt/cni/bin/. || true
done
