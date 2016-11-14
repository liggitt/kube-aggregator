package apiserver

import (
	"net/http"
	"sort"
	"strings"

	kapi "k8s.io/kubernetes/pkg/api"
	kapierrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apiserver"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"

	"github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	apifederationv1beta1 "github.com/openshift/kube-aggregator/pkg/apis/apifederation/v1beta1"
	listers "github.com/openshift/kube-aggregator/pkg/client/listers/apifederation/internalversion"
)

// apisHandler servers the `/apis` endpoint.
// This is registered as a filter so that it never collides with any explictly registered endpoints
type apisHandler struct {
	lister listers.APIServerLister

	delegate http.Handler
}

func (r *apisHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// if the URL is for OUR api group, serve it normally
	if strings.HasPrefix(req.URL.Path+"/", "/apis/"+apifederation.GroupName+"/") {
		r.delegate.ServeHTTP(w, req)
		return
	}
	// don't handle URLs that aren't /apis
	if req.URL.Path != "/apis" && req.URL.Path != "/apis/" {
		r.delegate.ServeHTTP(w, req)
		return
	}

	discoveryGroupList := &unversioned.APIGroupList{
		// always add OUR api group to the list first
		Groups: []unversioned.APIGroup{
			{
				Name: apifederation.GroupName,
				Versions: []unversioned.GroupVersionForDiscovery{
					{
						GroupVersion: apifederationv1beta1.SchemeGroupVersion.String(),
						Version:      apifederationv1beta1.SchemeGroupVersion.Version,
					},
				},
				PreferredVersion: unversioned.GroupVersionForDiscovery{
					GroupVersion: apifederationv1beta1.SchemeGroupVersion.String(),
					Version:      apifederationv1beta1.SchemeGroupVersion.Version,
				},
			},
		},
	}

	apiServers, err := r.lister.List(labels.Everything())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	apiServersByGroup := apifederation.SortByGroup(apiServers)
	for _, apiGroupServers := range apiServersByGroup {
		if len(apiGroupServers[0].Spec.Group) == 0 {
			continue
		}
		discoveryGroupList.Groups = append(discoveryGroupList.Groups, *newDiscoveryAPIGroup(apiGroupServers))
	}

	json, err := runtime.Encode(kapi.Codecs.LegacyCodec(), discoveryGroupList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(json); err != nil {
		panic(err)
	}
}

func newDiscoveryAPIGroup(apiServers []*apifederation.APIServer) *unversioned.APIGroup {
	serversByPriority := apifederation.ByPriority(apiServers)
	sort.Sort(serversByPriority)

	discoveryGroup := &unversioned.APIGroup{
		Name: apiServers[0].Spec.Group,
		PreferredVersion: unversioned.GroupVersionForDiscovery{
			GroupVersion: apiServers[0].Spec.Group + "/" + apiServers[0].Spec.Version,
			Version:      apiServers[0].Spec.Version,
		},
	}

	for _, apiServer := range serversByPriority {
		discoveryGroup.Versions = append(discoveryGroup.Versions,
			unversioned.GroupVersionForDiscovery{
				GroupVersion: apiServer.Spec.Group + "/" + apiServer.Spec.Version,
				Version:      apiServer.Spec.Version,
			},
		)
	}

	return discoveryGroup
}

// apiGroupHandler servers the `/apis/<group>` endpoint.
type apiGroupHandler struct {
	groupName string

	lister listers.APIServerLister
}

func (r *apiGroupHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	apiServers, err := r.lister.List(labels.Everything())
	if statusErr, ok := err.(*kapierrors.StatusError); ok && err != nil {
		apiserver.WriteRawJSON(int(statusErr.Status().Code), statusErr.Status(), w)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	apiServersForGroup := []*apifederation.APIServer{}
	for _, apiServer := range apiServers {
		if apiServer.Spec.Group == r.groupName {
			apiServersForGroup = append(apiServersForGroup, apiServer)
		}
	}

	if len(apiServersForGroup) == 0 {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	json, err := runtime.Encode(kapi.Codecs.LegacyCodec(), newDiscoveryAPIGroup(apiServersForGroup))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(json); err != nil {
		panic(err)
	}
}
