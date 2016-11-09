package api

import (
	api "github.com/openshift/kube-aggregator/pkg/api"
	pkg_api "k8s.io/kubernetes/pkg/api"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	watch "k8s.io/kubernetes/pkg/watch"
)

// APIServersGetter has a method to return a APIServerInterface.
// A group's client should implement this interface.
type APIServersGetter interface {
	APIServers() APIServerInterface
}

// APIServerInterface has methods to work with APIServer resources.
type APIServerInterface interface {
	Create(*api.APIServer) (*api.APIServer, error)
	Update(*api.APIServer) (*api.APIServer, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string) (*api.APIServer, error)
	List(opts v1.ListOptions) (*api.APIServerList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.APIServer, err error)
	APIServerExpansion
}

// aPIServers implements APIServerInterface
type aPIServers struct {
	client restclient.Interface
}

// newAPIServers returns a APIServers
func newAPIServers(c *PkgApiClient) *aPIServers {
	return &aPIServers{
		client: c.RESTClient(),
	}
}

// Create takes the representation of a aPIServer and creates it.  Returns the server's representation of the aPIServer, and an error, if there is any.
func (c *aPIServers) Create(aPIServer *api.APIServer) (result *api.APIServer, err error) {
	result = &api.APIServer{}
	err = c.client.Post().
		Resource("apiservers").
		Body(aPIServer).
		Do().
		Into(result)
	return
}

// Update takes the representation of a aPIServer and updates it. Returns the server's representation of the aPIServer, and an error, if there is any.
func (c *aPIServers) Update(aPIServer *api.APIServer) (result *api.APIServer, err error) {
	result = &api.APIServer{}
	err = c.client.Put().
		Resource("apiservers").
		Name(aPIServer.Name).
		Body(aPIServer).
		Do().
		Into(result)
	return
}

// Delete takes name of the aPIServer and deletes it. Returns an error if one occurs.
func (c *aPIServers) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("apiservers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *aPIServers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("apiservers").
		VersionedParams(&listOptions, pkg_api.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the aPIServer, and returns the corresponding aPIServer object, and an error if there is any.
func (c *aPIServers) Get(name string) (result *api.APIServer, err error) {
	result = &api.APIServer{}
	err = c.client.Get().
		Resource("apiservers").
		Name(name).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of APIServers that match those selectors.
func (c *aPIServers) List(opts v1.ListOptions) (result *api.APIServerList, err error) {
	result = &api.APIServerList{}
	err = c.client.Get().
		Resource("apiservers").
		VersionedParams(&opts, pkg_api.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested aPIServers.
func (c *aPIServers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Prefix("watch").
		Resource("apiservers").
		VersionedParams(&opts, pkg_api.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched aPIServer.
func (c *aPIServers) Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.APIServer, err error) {
	result = &api.APIServer{}
	err = c.client.Patch(pt).
		Resource("apiservers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
