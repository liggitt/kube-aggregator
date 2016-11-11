package fake

import (
	internalversion "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/apifederation/internalversion"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	core "k8s.io/kubernetes/pkg/client/testing/core"
)

type FakeApifederation struct {
	*core.Fake
}

func (c *FakeApifederation) APIServers() internalversion.APIServerInterface {
	return &FakeAPIServers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeApifederation) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
