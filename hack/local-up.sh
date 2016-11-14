#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"


ETCD_HOST=${ETCD_HOST:-127.0.0.1}
ETCD_PORT=${ETCD_PORT:-2379}

${OS_ROOT}/_output/local/bin/linux/amd64/kube-aggregator \
  --authentication-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --authorization-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --proxy-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --client-ca-file=/var/run/kubernetes/apiserver.crt \
  --tls-ca-file=/var/run/kubernetes/apiserver.crt \
  --secure-port=8444 \
  --etcd-servers="http://${ETCD_HOST}:${ETCD_PORT}" \
