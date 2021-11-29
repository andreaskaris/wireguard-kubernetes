#!/bin/bash

PodCIDR=$(kubectl get nodes $(cat /etc/hostname) -o json | jq '.spec.podCIDR' | sed 's/"//g')

cat <<EOF > /etc/cni/net.d/05-wireguard-cni.conflist
{
        "cniVersion": "0.3.1",
        "name": "wgcni",
        "plugins": [
                {
                        "type":"wgcni",
                        "mtu": 1500,
                        "ipam": {
                                "type": "host-local",
                                "dataDir": "/run/cni-ipam-state",
                                "routes": [
                                                { "dst": "0.0.0.0/0" }
                                ],
                                "ranges": [
                                        [ { "subnet": "10.244.2.0/24" } ]
                                ]
                        }
                },
                {
                        "type": "portmap",
                        "capabilities": {"portMappings": true},
                        "externalSetMarkChain": "KUBE-MARK-MASQ"
                }
        ]
}
EOF

for f in /cni-bin/*; do
	cp -n $f /opt/cni/bin/. || true
done
