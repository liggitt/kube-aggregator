package api

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

	Priority int64
}

type APIServerStatus struct {
	Group   string
	Version string
}

// +genclient=true

// APIServer is a logical top-level container for a set of origin resources
type APIServer struct {
	unversioned.TypeMeta
	kapi.ObjectMeta

	Spec   APIServerSpec
	Status APIServerStatus
}
