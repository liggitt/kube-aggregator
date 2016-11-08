package fake

import (
	api "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/discovery/api"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	core "k8s.io/kubernetes/pkg/client/testing/core"
)

type FakeDiscovery struct {
	*core.Fake
}

func (c *FakeDiscovery) APIServers(namespace string) api.APIServerInterface {
	return &FakeAPIServers{c, namespace}
}

// GetRESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeDiscovery) GetRESTClient() *restclient.RESTClient {
	return nil
}
