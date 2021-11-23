#!/bin/bash

cat <<'EOF' > /etc/cni/net.d/05-wireguard-cni.conf
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
			[ { "subnet": "10.244.0.0/24" } ]
		]
	},
	"mtu": 1500
}
EOF

for f in /cni-bin/*; do
	cp -n $f /opt/cni/bin/. || true
done

sleep infinity
