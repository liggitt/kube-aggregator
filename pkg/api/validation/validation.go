package validation

import (
	"fmt"
	"net"

	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/api/validation/path"
	utilvalidation "k8s.io/kubernetes/pkg/util/validation"
	"k8s.io/kubernetes/pkg/util/validation/field"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
)

func ValidateAPIServer(apiServer *discoveryapi.APIServer) field.ErrorList {
	requiredName := apiServer.Spec.Version + "." + apiServer.Spec.Group

	allErrs := validation.ValidateObjectMeta(&apiServer.ObjectMeta, false,
		func(name string, prefix bool) []string {
			if minimalFailures := path.IsValidPathSegmentName(name); len(minimalFailures) > 0 {
				return minimalFailures
			}
			// the name *must* be version.group
			if name != requiredName {
				return []string{fmt.Sprintf("must be `spec.version+\".\"+spec.group`: %q", requiredName)}
			}

			return []string{}
		},
		field.NewPath("metadata"))

	// in this case we all empty group
	if len(apiServer.Spec.Group) == 0 && apiServer.Spec.Version != "v1" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "group"), "only v1 may have an empty group and it better be legacy kube"))
	}
	if len(apiServer.Spec.Group) > 0 {
		for _, errString := range utilvalidation.IsDNS1123Subdomain(apiServer.Spec.Group) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "group"), apiServer.Spec.Group, errString))
		}
	}

	for _, errString := range utilvalidation.IsDNS1035Label(apiServer.Spec.Version) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "version"), apiServer.Spec.Version, errString))
	}

	if apiServer.Spec.Priority <= 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "priority"), apiServer.Spec.Priority, "priority must be positive"))

	}

	if len(apiServer.Spec.InternalHost) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "internalHost"), ""))
	} else {
		if _, _, err := net.SplitHostPort(apiServer.Spec.InternalHost); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "internalHost"), apiServer.Spec.InternalHost, err.Error()))
		}
	}

	return allErrs
}

func ValidateAPIServerUpdate(newAPIServer *discoveryapi.APIServer, oldAPIServer *discoveryapi.APIServer) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&newAPIServer.ObjectMeta, &oldAPIServer.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateAPIServer(newAPIServer)...)

	return allErrs
}
