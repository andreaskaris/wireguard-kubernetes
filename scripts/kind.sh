#!/bin/bash

CLUSTER_CONFIG_FILE=""

install_kind() {
	if  ! command -v kind &> /dev/null; then
		echo "Kind not found. Installing it."
		curl -Lo /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64
		chmod +x /usr/local/bin/kind
	fi
}

build_kind_custom_image() {
	go get k8s.io/kubernetes
}

write_cluster_config() {
	CLUSTER_CONFIG_FILE=$(mktemp)
	cat <<'EOF' > $CLUSTER_CONFIG_FILE
# three node (two workers) cluster config
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
#  disableDefaultCNI: true
  apiServerAddress: 0.0.0.0
  apiServerPort: 9999
nodes:
- role: control-plane
- role: control-plane
- role: control-plane
- role: worker
- role: worker
EOF
}

start_cluster() {
	kind create cluster --config $CLUSTER_CONFIG_FILE
}

show_cluster() {
	kind get clusters
	kind get nodes
	if  ! command -v kubectl &> /dev/null; then
		kubectl get nodes
	fi
}

wait_settle() {
	echo "Sleeping for 15 seconds"
	sleep 15
}

build_images() {
	echo "Building and loading wireguard-cni image"
	make -C container-images/wireguard-cni build-fedora
	kind load docker-image wireguard-cni
	echo "Building and loading wireguard-wgk8s image"
	make -C container-images/wireguard-wgk8s build-fedora
	kind load docker-image wireguard-wgk8s
}

deploy_wireguard_kubernetes() {
	echo "Deploying wireguard kubernetes"
	kubectl apply -f custom-resources/kind/namespace.yaml
	kubectl apply -f custom-resources/kind/rolebindings.yaml
	kubectl apply -f custom-resources/kind/daemonset.yaml
}

delete_cluster() {
	kind delete cluster
}

if [ "$1" == "--delete" ]; then
	delete_cluster
	exit 1
fi

install_kind
build_kind_custom_image
write_cluster_config
start_cluster
show_cluster
wait_settle
build_images
deploy_wireguard_kubernetes
