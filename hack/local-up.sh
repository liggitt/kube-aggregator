#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"


ETCD_HOST=${ETCD_HOST:-127.0.0.1}
ETCD_PORT=${ETCD_PORT:-2379}

${OS_ROOT}/_output/local/bin/linux/amd64/kube-aggregator \
  --kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --client-ca-file=/var/run/kubernetes/apiserver.crt \
  --etcd-servers="http://${ETCD_HOST}:${ETCD_PORT}" \
