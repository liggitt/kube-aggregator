package apifederation

import (
	fmt "fmt"
	api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	registered "k8s.io/kubernetes/pkg/apimachinery/registered"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	serializer "k8s.io/kubernetes/pkg/runtime/serializer"
)


type .ApifederationInterface interface {
    RESTClient() restclient.Interface
     APIServersGetter
    
}

// .ApifederationClient is used to interact with features provided by the k8s.io/kubernetes/pkg/apimachinery/registered.Group group.
type .ApifederationClient struct {
	restClient restclient.Interface
}

func (c *.ApifederationClient) APIServers() APIServerInterface {
	return newAPIServers(c)
}

// NewForConfig creates a new .ApifederationClient for the given config.
func NewForConfig(c *restclient.Config) (*.ApifederationClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &.ApifederationClient{client}, nil
}

// NewForConfigOrDie creates a new .ApifederationClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *.ApifederationClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new .ApifederationClient for the given RESTClient.
func New(c restclient.Interface) *.ApifederationClient {
	return &.ApifederationClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	gv, err := unversioned.ParseGroupVersion("apifederation.openshift.io/apifederation")
	if err != nil {
		return err
	}
	// if apifederation.openshift.io/apifederation is not enabled, return an error
	if ! registered.IsEnabledVersion(gv) {
		return fmt.Errorf("apifederation.openshift.io/apifederation is not enabled")
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
func (c *.ApifederationClient) RESTClient() restclient.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
