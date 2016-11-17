package apifederation

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// APIServerList is a list of APIServer objects.
type APIServerList struct {
	unversioned.TypeMeta
	unversioned.ListMeta

	Items []APIServer
}

type APIServerSpec struct {
	InternalHost string
	Prefix       string
	Group        string
	Version      string

	InsecureSkipTLSVerify bool
	CABundle              []byte

	Priority int64
}

type APIServerStatus struct {
}

type APIResource struct {
	Name       string
	Namespaced bool
	Kind       string

	SubResources []APISubResource
}

type APISubResource struct {
	Name string
	Kind string
}

// +genclient=true
// +nonNamespaced=true

// APIServer is a logical top-level container for a set of origin resources
type APIServer struct {
	unversioned.TypeMeta
	kapi.ObjectMeta

	Spec   APIServerSpec
	Status APIServerStatus
}
