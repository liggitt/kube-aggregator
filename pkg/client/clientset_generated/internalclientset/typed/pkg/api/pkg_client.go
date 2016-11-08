package api

import (
	fmt "fmt"
	api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	registered "k8s.io/kubernetes/pkg/apimachinery/registered"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	serializer "k8s.io/kubernetes/pkg/runtime/serializer"
)

type PkgApiInterface interface {
	RESTClient() restclient.Interface
	APIServersGetter
}

// PkgApiClient is used to interact with features provided by the k8s.io/kubernetes/pkg/apimachinery/registered.Group group.
type PkgApiClient struct {
	restClient restclient.Interface
}

func (c *PkgApiClient) APIServers(namespace string) APIServerInterface {
	return newAPIServers(c, namespace)
}

// NewForConfig creates a new PkgApiClient for the given config.
func NewForConfig(c *restclient.Config) (*PkgApiClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PkgApiClient{client}, nil
}

// NewForConfigOrDie creates a new PkgApiClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *PkgApiClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new PkgApiClient for the given RESTClient.
func New(c restclient.Interface) *PkgApiClient {
	return &PkgApiClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	gv, err := unversioned.ParseGroupVersion("pkg/api")
	if err != nil {
		return err
	}
	// if pkg/api is not enabled, return an error
	if !registered.IsEnabledVersion(gv) {
		return fmt.Errorf("pkg/api is not enabled")
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}
	copyGroupVersion := gv
	config.GroupVersion = &copyGroupVersion

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *PkgApiClient) RESTClient() restclient.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
