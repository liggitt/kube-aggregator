package fake

import (
	apifederation "github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	core "k8s.io/kubernetes/pkg/client/testing/core"
	labels "k8s.io/kubernetes/pkg/labels"
	watch "k8s.io/kubernetes/pkg/watch"
)

// FakeAPIServers implements APIServerInterface
type FakeAPIServers struct {
	Fake *Fake.Apifederation
}

var apiserversResource = unversioned.GroupVersionResource{Group: "apifederation.openshift.io", Version: "apifederation", Resource: "apiservers"}

func (c *FakeAPIServers) Create(aPIServer *apifederation.APIServer) (result *apifederation.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootCreateAction(apiserversResource, aPIServer), &apifederation.APIServer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apifederation.APIServer), err
}

func (c *FakeAPIServers) Update(aPIServer *apifederation.APIServer) (result *apifederation.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootUpdateAction(apiserversResource, aPIServer), &apifederation.APIServer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apifederation.APIServer), err
}

func (c *FakeAPIServers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewRootDeleteAction(apiserversResource, name), &apifederation.APIServer{})
	return err
}

func (c *FakeAPIServers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := core.NewRootDeleteCollectionAction(apiserversResource, listOptions)

	_, err := c.Fake.Invokes(action, &apifederation.APIServerList{})
	return err
}

func (c *FakeAPIServers) Get(name string) (result *apifederation.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootGetAction(apiserversResource, name), &apifederation.APIServer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apifederation.APIServer), err
}

func (c *FakeAPIServers) List(opts v1.ListOptions) (result *apifederation.APIServerList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootListAction(apiserversResource, opts), &apifederation.APIServerList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &apifederation.APIServerList{}
	for _, item := range obj.(*apifederation.APIServerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aPIServers.
func (c *FakeAPIServers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewRootWatchAction(apiserversResource, opts))
}

// Patch applies the patch and returns the patched aPIServer.
func (c *FakeAPIServers) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *apifederation.APIServer, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootPatchSubresourceAction(apiserversResource, name, data, subresources...), &apifederation.APIServer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apifederation.APIServer), err
}
