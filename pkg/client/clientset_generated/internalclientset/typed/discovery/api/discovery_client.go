package api

import (
	api "k8s.io/kubernetes/pkg/api"
	registered "k8s.io/kubernetes/pkg/apimachinery/registered"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	serializer "k8s.io/kubernetes/pkg/runtime/serializer"
)

type DiscoveryInterface interface {
	GetRESTClient() *restclient.RESTClient
	APIServersGetter
}

// DiscoveryClient is used to interact with features provided by the Discovery group.
type DiscoveryClient struct {
	*restclient.RESTClient
}

func (c *DiscoveryClient) APIServers(namespace string) APIServerInterface {
	return newAPIServers(c, namespace)
}

// NewForConfig creates a new DiscoveryClient for the given config.
func NewForConfig(c *restclient.Config) (*DiscoveryClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &DiscoveryClient{client}, nil
}

// NewForConfigOrDie creates a new DiscoveryClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *DiscoveryClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new DiscoveryClient for the given RESTClient.
func New(c *restclient.RESTClient) *DiscoveryClient {
	return &DiscoveryClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	// if discovery group is not registered, return an error
	g, err := registered.Group("discovery")
	if err != nil {
		return err
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}
	// TODO: Unconditionally set the config.Version, until we fix the config.
	//if config.Version == "" {
	copyGroupVersion := g.GroupVersion
	config.GroupVersion = &copyGroupVersion
	//}

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return nil
}

// GetRESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *DiscoveryClient) GetRESTClient() *restclient.RESTClient {
	if c == nil {
		return nil
	}
	return c.RESTClient
}
