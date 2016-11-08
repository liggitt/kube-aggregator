package v1beta1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch/versioned"
)

const GroupName = "apifederation.openshift.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: "v1beta1"}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&APIServer{},
		&APIServerList{},

		&v1.ListOptions{},
		&v1.DeleteOptions{},
		&v1.ExportOptions{},
	)
	versioned.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func (obj *APIServer) GetObjectKind() unversioned.ObjectKind     { return &obj.TypeMeta }
func (obj *APIServerList) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }
