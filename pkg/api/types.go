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

type APIServerConditionType string

const (
	APIServerAvailable APIServerConditionType = "Available"
)

type APIServerCondition struct {
	Type               APIServerConditionType
	Status             kapi.ConditionStatus
	LastProbeTime      unversioned.Time
	LastTransitionTime unversioned.Time
	Reason             string
	Message            string
}

type APIServerStatus struct {
	Conditions []APIServerCondition

	Group     string
	Version   string
	Resources []APIResource
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
