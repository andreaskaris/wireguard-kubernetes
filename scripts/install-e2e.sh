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
		# https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md
		make WHAT=test/e2e/e2e.test GOGCFLAGS="all=-N -l" GOLDFLAGS=""
		\cp ./_output/bin/e2e.test /usr/local/bin/e2e.test
	else 
		# from https://github.com/ovn-org/ovn-kubernetes/blob/master/test/scripts/install-kind.sh
		echo "Downloading e2e tests"
		curl -L "https://github.com/andreaskaris/e2e-binaries/blob/master/e2e.tag.gz?raw=true" -o e2e.tag.gz
		tar xvzf e2e.tag.gz?
		mv ./e2e.test /usr/local/bin/e2e.test
	fi
fi
