package api

import (
	api "k8s.io/kubernetes/pkg/api"
	registered "k8s.io/kubernetes/pkg/apimachinery/registered"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	serializer "k8s.io/kubernetes/pkg/runtime/serializer"
)

type PkgInterface interface {
	GetRESTClient() *restclient.RESTClient
	APIServersGetter
}

// PkgClient is used to interact with features provided by the Pkg group.
type PkgClient struct {
	*restclient.RESTClient
}

func (c *PkgClient) APIServers(namespace string) APIServerInterface {
	return newAPIServers(c, namespace)
}

// NewForConfig creates a new PkgClient for the given config.
func NewForConfig(c *restclient.Config) (*PkgClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PkgClient{client}, nil
}

// NewForConfigOrDie creates a new PkgClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *PkgClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new PkgClient for the given RESTClient.
func New(c *restclient.RESTClient) *PkgClient {
	return &PkgClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	// if pkg group is not registered, return an error
	g, err := registered.Group("pkg")
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
func (c *PkgClient) GetRESTClient() *restclient.RESTClient {
	if c == nil {
		return nil
	}
	return c.RESTClient
}
