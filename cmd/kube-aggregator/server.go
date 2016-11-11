package main

import (
	"os"
	"runtime"

	"k8s.io/kubernetes/pkg/util/logs"

	"github.com/openshift/kube-aggregator/pkg/cmd/server"

	// install all APIs
	_ "github.com/openshift/kube-aggregator/pkg/apis/apifederation/install"
	_ "k8s.io/kubernetes/pkg/api/install"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	// defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	// defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	cmd := server.NewCommandStartDiscoveryServer(os.Stdout)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
