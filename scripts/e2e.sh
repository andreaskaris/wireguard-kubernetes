#!/bin/bash -x
# https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

$DIR/install-e2e.sh

echo "Running network conformance tests"
export KUBECONFIG="${HOME}/.kube/config"

if [ "$1" == "" ]; then
	/usr/local/bin/e2e.test -context kind-kind -ginkgo.focus="\[sig-network\].*Conformance" -num-nodes 3
else
	/usr/local/bin/e2e.test -context kind-kind -ginkgo.focus="$1" -num-nodes 3
fi
