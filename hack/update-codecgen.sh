#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

export CODECGEN_GENERATED_FILES="
pkg/apis/apifederation/v1beta1/types.generated.go
"

export CODECGEN_PREFIX=github.com/openshift/kube-aggregator

${OS_ROOT}/hack/run-codecgen.sh
