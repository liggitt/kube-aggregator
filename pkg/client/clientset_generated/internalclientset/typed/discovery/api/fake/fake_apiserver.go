package fake

import (
	api "github.com/openshift/kube-aggregator/pkg/api"
	pkg_api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	core "k8s.io/kubernetes/pkg/client/testing/core"
	labels "k8s.io/kubernetes/pkg/labels"
	watch "k8s.io/kubernetes/pkg/watch"
)

// FakeAPIServers implements APIServerInterface
type FakeAPIServers struct {
	Fake *FakeDiscovery
	ns   string
}

var apiserversResource = unversioned.GroupVersionResource{Group: "discovery", Version: "api", Resource: "apiservers"}

func (c *FakeAPIServers) Create(aPIServer *api.APIServer) (result *api.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewCreateAction(apiserversResource, c.ns, aPIServer), &api.APIServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.APIServer), err
}

func (c *FakeAPIServers) Update(aPIServer *api.APIServer) (result *api.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewUpdateAction(apiserversResource, c.ns, aPIServer), &api.APIServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.APIServer), err
}

func (c *FakeAPIServers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewDeleteAction(apiserversResource, c.ns, name), &api.APIServer{})

	return err
}

func (c *FakeAPIServers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := core.NewDeleteCollectionAction(apiserversResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &api.APIServerList{})
	return err
}

func (c *FakeAPIServers) Get(name string) (result *api.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewGetAction(apiserversResource, c.ns, name), &api.APIServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.APIServer), err
}

func (c *FakeAPIServers) List(opts v1.ListOptions) (result *api.APIServerList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewListAction(apiserversResource, c.ns, opts), &api.APIServerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &api.APIServerList{}
	for _, item := range obj.(*api.APIServerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aPIServers.
func (c *FakeAPIServers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewWatchAction(apiserversResource, c.ns, opts))

}

// Patch applies the patch and returns the patched aPIServer.
func (c *FakeAPIServers) Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewPatchSubresourceAction(apiserversResource, c.ns, name, data, subresources...), &api.APIServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.APIServer), err
}