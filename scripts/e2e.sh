#!/bin/bash -x
# https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

echo "Building e2e tests - this can take a while"
pushd /tmp
git clone https://github.com/kubernetes/kubernetes
cd kubernetes
git checkout -t origin/release-1.21
make WHAT="test/e2e/e2e.test"

echo "Running network conformance tests"
export KUBECONFIG="${HOME}/.kube/config"
./_output/bin/e2e.test -context kind-kind -ginkgo.focus="\[sig-network\].*Conformance" -num-nodes 3
