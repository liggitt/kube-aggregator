package fake

import (
	core "k8s.io/kubernetes/pkg/client/testing/core"
	apifederation "github.com/openshift/kube-aggregator/pkg/apis/pkg/client/clientset_generated/internalclientset/typed/apifederation"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
)


type Fake.Apifederation struct {
	*core.Fake
}

func (c *Fake.Apifederation) APIServers() apifederation.APIServerInterface {
	return &FakeAPIServers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *Fake.Apifederation) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
