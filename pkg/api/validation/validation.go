package validation

import (
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/util/validation/field"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
)

func ValidateAPIServerName(name string, prefix bool) []string {
	if reasons := validation.ValidateNamespaceName(name, false); len(reasons) != 0 {
		return reasons
	}

	return nil
}

func ValidateAPIServer(apiServer *discoveryapi.APIServer) field.ErrorList {
	result := validation.ValidateObjectMeta(&apiServer.ObjectMeta, false, ValidateAPIServerName, field.NewPath("metadata"))

	return result
}

func ValidateAPIServerUpdate(newAPIServer *discoveryapi.APIServer, oldAPIServer *discoveryapi.APIServer) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&newAPIServer.ObjectMeta, &oldAPIServer.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateAPIServer(newAPIServer)...)

	return allErrs
}
