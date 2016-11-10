package fake

import (
	api "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/pkg/api"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	core "k8s.io/kubernetes/pkg/client/testing/core"
)

type FakePkgApi struct {
	*core.Fake
}

func (c *FakePkgApi) APIServers() api.APIServerInterface {
	return &FakeAPIServers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakePkgApi) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
