#!/bin/bash

cat <<'EOF' > /etc/cni/net.d/05-wireguard-cni.conf
{
  "cniVersion": "0.3.1",
  "name": "debug",
  "type": "debug",
  "cniOutput": "/tmp/cni_output.txt"
}
EOF

cp /cni-bin/* /opt/cni/bin/.

sleep infinity
