#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm -f "${CLIENTGEN:-}"
  rm -f "${listergen:-}"
}
trap cleanup EXIT

echo "Building client-gen"
CLIENTGEN="${PWD}/client-gen-binary"
go build -o "${CLIENTGEN}" ./vendor/k8s.io/kubernetes/cmd/libs/go2idl/client-gen

PREFIX=github.com/openshift/kube-aggregator/pkg/apis
INPUT_BASE="--input-base ${PREFIX}"
INPUT_APIS=(
apifederation/
apifederation/v1beta1
)
INPUT="--input ${INPUT_APIS[@]}"
CLIENTSET_PATH="--clientset-path github.com/openshift/kube-aggregator/pkg/client/clientset_generated"
BOILERPLATE="--go-header-file ${OS_ROOT}/hack/boilerplate.txt"

${CLIENTGEN} ${INPUT_BASE} ${INPUT} ${CLIENTSET_PATH} ${BOILERPLATE}


echo "Building lister-gen"
listergen="${PWD}/lister-gen"
go build -o "${listergen}" ./vendor/k8s.io/kubernetes/cmd/libs/go2idl/lister-gen

LISTER_INPUT="--input-dirs github.com/openshift/kube-aggregator/pkg/apis/apifederation --input-dirs github.com/openshift/kube-aggregator/pkg/apis/apifederation/v1beta1"
LISTER_PATH="--output-package github.com/openshift/kube-aggregator/pkg/client/listers"
${listergen} ${LISTER_INPUT} ${LISTER_PATH} ${BOILERPLATE}