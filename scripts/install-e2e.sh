#!/bin/bash -x
# https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

echo "Moving to /tmp"
pushd /tmp

if ! [ -f /usr/local/bin/e2e.test ]; then
	if [ -f /etc/redhat-release ]; then
		echo "Building e2e tests - this can take a while"
		git clone https://github.com/kubernetes/kubernetes
		cd kubernetes
		git checkout -t origin/release-1.21
		make WHAT="test/e2e/e2e.test"
		\cp ./_output/bin/e2e.test /usr/local/bin/e2e.test
	else 
		# from https://github.com/ovn-org/ovn-kubernetes/blob/master/test/scripts/install-kind.sh
		echo "Downloading e2e tests"
		curl -L "https://github.com/trozet/ovnFiles/blob/master/kubernetes-test-linux-v1.21.0-alpha.0.341%2B46d481b4556e33.tar.gz?raw=true" -o kubernetes-test-linux-amd64.tar.gz
		tar xvzf kubernetes-test-linux-amd64.tar.gz
		mv ./e2e.test /usr/local/bin/e2e.test
	fi
fi
